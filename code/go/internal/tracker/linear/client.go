// Package linear implements the tracker.Client interface for Linear's GraphQL API.
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/oneneural/tempad/internal/tracker"
)

const (
	defaultPageSize = 50
	defaultTimeout  = 30 * time.Second
)

// LinearClient communicates with Linear's GraphQL API.
type LinearClient struct {
	endpoint       string
	apiKey         string
	projectSlug    string
	identity       string // email
	userID         string // resolved from identity via API (cached)
	httpClient     *http.Client
	activeStates   []string
	terminalStates []string
}

// Config holds the configuration needed to construct a LinearClient.
type Config struct {
	Endpoint       string
	APIKey         string
	ProjectSlug    string
	Identity       string // email for identity resolution
	ActiveStates   []string
	TerminalStates []string
}

// NewLinearClient creates a new Linear API client. It does NOT resolve the
// user identity at construction — call ResolveIdentity separately.
func NewLinearClient(cfg Config) *LinearClient {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.linear.app/graphql"
	}

	return &LinearClient{
		endpoint:       endpoint,
		apiKey:         cfg.APIKey,
		projectSlug:    cfg.ProjectSlug,
		identity:       cfg.Identity,
		httpClient:     &http.Client{Timeout: defaultTimeout},
		activeStates:   cfg.ActiveStates,
		terminalStates: cfg.TerminalStates,
	}
}

// do sends a GraphQL request to Linear and unmarshals the response into result.
// It checks for HTTP errors, GraphQL errors, and rate limiting.
func (c *LinearClient) do(ctx context.Context, query string, vars map[string]any, result any) error {
	body, err := json.Marshal(graphqlRequest{
		Query:     query,
		Variables: vars,
	})
	if err != nil {
		return &tracker.APIRequestError{Message: "marshal request", Cause: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return &tracker.APIRequestError{Message: "create request", Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &tracker.APIRequestError{Message: "execute request", Cause: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tracker.APIRequestError{Message: "read response", Cause: err}
	}

	// Rate limit check.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 60 // default
		if val := resp.Header.Get("Retry-After"); val != "" {
			if parsed, parseErr := strconv.Atoi(val); parseErr == nil {
				retryAfter = parsed
			}
		}
		return &tracker.RateLimitError{RetryAfterSecs: retryAfter}
	}

	// Non-200 HTTP errors.
	if resp.StatusCode != http.StatusOK {
		return &tracker.APIStatusError{
			StatusCode: resp.StatusCode,
			Body:       truncate(string(respBody), 500),
		}
	}

	// Parse response — Linear returns 200 even for GraphQL errors.
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphqlError  `json:"errors,omitempty"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return &tracker.APIRequestError{Message: "unmarshal response", Cause: err}
	}

	// Check GraphQL-level errors.
	if len(envelope.Errors) > 0 {
		msgs := make([]string, len(envelope.Errors))
		for i, e := range envelope.Errors {
			msgs[i] = e.Message
		}
		return &tracker.APIErrorsError{Errors: msgs}
	}

	// Unmarshal data into the caller's result type.
	if result != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return &tracker.APIRequestError{
				Message: "unmarshal data",
				Cause:   fmt.Errorf("into %T: %w", result, err),
			}
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
