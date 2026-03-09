package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SetupResult holds the validated configuration from the setup wizard.
type SetupResult struct {
	APIKey        string
	Identity      string
	IDECommand    string
	AgentCommand  string
	WorkspaceRoot string
}

const defaultLinearEndpoint = "https://api.linear.app/graphql"

// ValidateAPIKey checks that a Linear API key is valid by making a test
// API call to the Linear GraphQL endpoint.
func ValidateAPIKey(ctx context.Context, apiKey string) error {
	return validateAPIKeyWithEndpoint(ctx, apiKey, defaultLinearEndpoint)
}

func validateAPIKeyWithEndpoint(ctx context.Context, apiKey, endpoint string) error {
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	body, err := json.Marshal(map[string]string{
		"query": "{ viewer { id } }",
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("API key invalid (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected API response (HTTP %d)", resp.StatusCode)
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("API key invalid: %s", envelope.Errors[0].Message)
	}

	return nil
}

// ValidateIdentity checks that a Linear email resolves to a user by querying
// the Linear API.
func ValidateIdentity(ctx context.Context, apiKey, email string) error {
	return validateIdentityWithEndpoint(ctx, apiKey, email, defaultLinearEndpoint)
}

func validateIdentityWithEndpoint(ctx context.Context, apiKey, email, endpoint string) error {
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("identity cannot be empty")
	}

	query := `query UserByEmail($email: String!) {
		users(filter: { email: { eq: $email } }) {
			nodes { id email }
		}
	}`

	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]any{"email": email},
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed (HTTP %d)", resp.StatusCode)
	}

	var envelope struct {
		Data struct {
			Users struct {
				Nodes []struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"nodes"`
			} `json:"users"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("API error: %s", envelope.Errors[0].Message)
	}
	if len(envelope.Data.Users.Nodes) == 0 {
		return fmt.Errorf("no Linear user found for email %q", email)
	}

	return nil
}

// LookupBinary checks whether a command binary exists on PATH.
// Returns the resolved path or an error if not found.
func LookupBinary(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("binary name cannot be empty")
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%q not found on PATH", name)
	}
	return path, nil
}

// DefaultWorkspaceRoot returns the default workspace root path.
func DefaultWorkspaceRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "tempad_workspaces")
	}
	return filepath.Join(home, ".tempad", "workspaces")
}

// EnsureDirectory creates a directory and all parents if it doesn't exist.
func EnsureDirectory(path string) error {
	path = ExpandHome(path)
	return os.MkdirAll(path, 0755)
}

// WriteUserConfig writes a UserConfig to the given path as YAML.
func WriteUserConfig(path string, cfg *UserConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	header := "# TEMPAD User Configuration\n# Generated by `tempad setup`\n# This file stores personal preferences. Team settings live in WORKFLOW.md.\n\n"
	content := header + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// KnownIDEs is the list of IDE commands presented during setup.
var KnownIDEs = []struct {
	Name    string
	Command string
}{
	{"VS Code", "code"},
	{"Cursor", "cursor"},
	{"Zed", "zed"},
	{"IntelliJ IDEA", "idea"},
	{"WebStorm", "webstorm"},
}

// KnownAgents is the list of agent commands presented during setup.
var KnownAgents = []struct {
	Name    string
	Command string
}{
	{"Claude Code", "claude"},
	{"Codex", "codex"},
	{"OpenCode", "opencode"},
}
