package linear

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/oneneural/tempad/internal/tracker"
)

func TestDo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers.
		if r.Header.Get("Authorization") != "test-key" {
			t.Error("missing or wrong Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issue": map[string]any{
					"id":    "issue-1",
					"title": "Test Issue",
				},
			},
		})
	}))
	defer srv.Close()

	c := NewLinearClient(Config{
		Endpoint: srv.URL,
		APIKey:   "test-key",
	})

	var result struct {
		Issue struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"issue"`
	}

	err := c.do(context.Background(), "query { issue { id title } }", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Issue.ID != "issue-1" {
		t.Errorf("expected id=issue-1, got %s", result.Issue.ID)
	}
	if result.Issue.Title != "Test Issue" {
		t.Errorf("expected title=Test Issue, got %s", result.Issue.Title)
	}
}

func TestDo_GraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Linear returns 200 with GraphQL errors.
		json.NewEncoder(w).Encode(map[string]any{
			"data": nil,
			"errors": []map[string]any{
				{"message": "Field 'foo' not found"},
				{"message": "Invalid query"},
			},
		})
	}))
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	err := c.do(context.Background(), "query { foo }", nil, nil)

	var target *tracker.APIErrorsError
	if !errors.As(err, &target) {
		t.Fatalf("expected APIErrorsError, got %T: %v", err, err)
	}
	if len(target.Errors) != 2 {
		t.Errorf("expected 2 GraphQL errors, got %d", len(target.Errors))
	}
}

func TestDo_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "bad-key"})
	err := c.do(context.Background(), "query {}", nil, nil)

	var target *tracker.APIStatusError
	if !errors.As(err, &target) {
		t.Fatalf("expected APIStatusError, got %T: %v", err, err)
	}
	if target.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", target.StatusCode)
	}
}

func TestDo_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	err := c.do(context.Background(), "query {}", nil, nil)

	var target *tracker.RateLimitError
	if !errors.As(err, &target) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if target.RetryAfterSecs != 30 {
		t.Errorf("expected RetryAfterSecs=30, got %d", target.RetryAfterSecs)
	}
}

func TestDo_NetworkError(t *testing.T) {
	c := NewLinearClient(Config{Endpoint: "http://localhost:1", APIKey: "key"})
	err := c.do(context.Background(), "query {}", nil, nil)

	var target *tracker.APIRequestError
	if !errors.As(err, &target) {
		t.Fatalf("expected APIRequestError, got %T: %v", err, err)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response — should be cancelled.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := c.do(ctx, "query {}", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}

	var target *tracker.APIRequestError
	if !errors.As(err, &target) {
		t.Fatalf("expected APIRequestError, got %T: %v", err, err)
	}
}

func TestFetchAll_ThreePages(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		json.NewDecoder(r.Body).Decode(&req)

		page := requestCount.Add(1)
		var resp map[string]any

		switch page {
		case 1:
			resp = map[string]any{
				"data": map[string]any{
					"issues": map[string]any{
						"nodes": []map[string]any{
							{"id": "1", "identifier": "T-1", "title": "Issue 1"},
							{"id": "2", "identifier": "T-2", "title": "Issue 2"},
						},
						"pageInfo": map[string]any{
							"hasNextPage": true,
							"endCursor":   "cursor-1",
						},
					},
				},
			}
		case 2:
			// Verify cursor was passed.
			if req.Variables["after"] != "cursor-1" {
				t.Errorf("expected after=cursor-1, got %v", req.Variables["after"])
			}
			resp = map[string]any{
				"data": map[string]any{
					"issues": map[string]any{
						"nodes": []map[string]any{
							{"id": "3", "identifier": "T-3", "title": "Issue 3"},
						},
						"pageInfo": map[string]any{
							"hasNextPage": true,
							"endCursor":   "cursor-2",
						},
					},
				},
			}
		case 3:
			if req.Variables["after"] != "cursor-2" {
				t.Errorf("expected after=cursor-2, got %v", req.Variables["after"])
			}
			resp = map[string]any{
				"data": map[string]any{
					"issues": map[string]any{
						"nodes":    []map[string]any{},
						"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
					},
				},
			}
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})

	type simpleIssue struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
	}
	type issuesResp struct {
		Issues struct {
			Nodes    []simpleIssue `json:"nodes"`
			PageInfo pageInfo      `json:"pageInfo"`
		} `json:"issues"`
	}

	results, err := fetchAll[issuesResp, simpleIssue](
		context.Background(),
		c,
		"query ($first: Int!, $after: String) { issues(first: $first, after: $after) { nodes { id identifier title } pageInfo { hasNextPage endCursor } } }",
		nil,
		func(data *issuesResp) page[simpleIssue] {
			return page[simpleIssue]{
				Nodes:    data.Issues.Nodes,
				PageInfo: data.Issues.PageInfo,
			}
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Identifier != "T-1" {
		t.Errorf("expected first result T-1, got %s", results[0].Identifier)
	}
	if results[2].Identifier != "T-3" {
		t.Errorf("expected third result T-3, got %s", results[2].Identifier)
	}

	if requestCount.Load() != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount.Load())
	}
}
