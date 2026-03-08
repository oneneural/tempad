package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockServer creates a test server that returns the given response for any request.
func mockServer(t *testing.T, response any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	}))
}

func TestResolveIdentity(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"data": map[string]any{
			"users": map[string]any{
				"nodes": []map[string]any{
					{"id": "user-uuid-123", "email": "dev@example.com"},
				},
			},
		},
	})
	defer srv.Close()

	c := NewLinearClient(Config{
		Endpoint: srv.URL,
		APIKey:   "key",
		Identity: "dev@example.com",
	})

	err := c.ResolveIdentity(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.userID != "user-uuid-123" {
		t.Errorf("expected userID=user-uuid-123, got %s", c.userID)
	}
}

func TestResolveIdentity_NoUser(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"data": map[string]any{
			"users": map[string]any{"nodes": []any{}},
		},
	})
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key", Identity: "unknown@example.com"})
	err := c.ResolveIdentity(context.Background())
	if err == nil {
		t.Fatal("expected error for unknown email")
	}
}

func TestFetchCandidateIssues(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req graphqlRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Both candidate and assignedToMe queries return single pages.
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{
						{
							"id": "issue-" + string(rune('0'+callCount)), "identifier": "T-" + string(rune('0'+callCount)),
							"title": "Issue", "state": map[string]any{"name": "Todo"},
							"labels": map[string]any{"nodes": []any{}},
							"relations": map[string]any{"nodes": []any{}},
							"createdAt": "2026-03-08T12:00:00Z", "updatedAt": "2026-03-08T12:00:00Z",
						},
					},
					"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewLinearClient(Config{
		Endpoint:    srv.URL,
		APIKey:      "key",
		ProjectSlug: "my-project",
		ActiveStates: []string{"Todo"},
	})
	c.userID = "user-1" // Pre-set to avoid identity resolution.

	issues, err := c.FetchCandidateIssues(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should make 2 calls: unassigned + assigned to me.
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestFetchCandidateIssues_Deduplication(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Both queries return the same issue — should be deduplicated.
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{
						{
							"id": "same-issue", "identifier": "T-1", "title": "Dup",
							"state": map[string]any{"name": "Todo"},
							"labels": map[string]any{"nodes": []any{}},
							"relations": map[string]any{"nodes": []any{}},
							"createdAt": "2026-03-08T12:00:00Z", "updatedAt": "2026-03-08T12:00:00Z",
						},
					},
					"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewLinearClient(Config{
		Endpoint:    srv.URL, APIKey: "key", ProjectSlug: "proj",
		ActiveStates: []string{"Todo"},
	})
	c.userID = "user-1"

	issues, err := c.FetchCandidateIssues(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 deduplicated issue, got %d", len(issues))
	}
}

func TestFetchIssuesByStates(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"data": map[string]any{
			"issues": map[string]any{
				"nodes": []map[string]any{
					{
						"id": "done-1", "identifier": "T-10", "title": "Done Issue",
						"state": map[string]any{"name": "Done"},
						"labels": map[string]any{"nodes": []any{}},
						"relations": map[string]any{"nodes": []any{}},
						"createdAt": "2026-03-08T12:00:00Z", "updatedAt": "2026-03-08T12:00:00Z",
					},
				},
				"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
			},
		},
	})
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key", ProjectSlug: "proj"})
	issues, err := c.FetchIssuesByStates(context.Background(), []string{"Done", "Closed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].State != "Done" {
		t.Errorf("expected state=Done, got %s", issues[0].State)
	}
}

func TestFetchIssueStatesByIDs(t *testing.T) {
	// FetchIssueStatesByIDs now fetches each issue individually via singleIssueQuery.
	// Use a request-aware mock that returns different responses per issue ID.
	callCount := 0
	responses := []map[string]any{
		{
			"data": map[string]any{
				"issue": map[string]any{
					"id": "id-1", "identifier": "T-1", "title": "Task 1",
					"state":     map[string]any{"name": "In Progress"},
					"labels":    map[string]any{"nodes": []any{}},
					"relations": map[string]any{"nodes": []any{}},
					"createdAt": "2026-03-08T12:00:00Z", "updatedAt": "2026-03-08T12:00:00Z",
				},
			},
		},
		{
			"data": map[string]any{
				"issue": map[string]any{
					"id": "id-2", "identifier": "T-2", "title": "Task 2",
					"state":     map[string]any{"name": "Done"},
					"labels":    map[string]any{"nodes": []any{}},
					"relations": map[string]any{"nodes": []any{}},
					"createdAt": "2026-03-08T12:00:00Z", "updatedAt": "2026-03-08T12:00:00Z",
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := callCount
		if idx >= len(responses) {
			idx = len(responses) - 1
		}
		callCount++
		json.NewEncoder(w).Encode(responses[idx])
	}))
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	states, err := c.FetchIssueStatesByIDs(context.Background(), []string{"id-1", "id-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if states["id-1"] != "In Progress" {
		t.Errorf("id-1: got %q", states["id-1"])
	}
	if states["id-2"] != "Done" {
		t.Errorf("id-2: got %q", states["id-2"])
	}
}

func TestFetchIssue(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"id": "uuid-1", "identifier": "ABC-123", "title": "Test",
				"state": map[string]any{"name": "Todo"},
				"labels": map[string]any{"nodes": []any{}},
				"relations": map[string]any{"nodes": []any{}},
				"createdAt": "2026-03-08T12:00:00Z", "updatedAt": "2026-03-08T12:00:00Z",
			},
		},
	})
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	issue, err := c.FetchIssue(context.Background(), "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Identifier != "ABC-123" {
		t.Errorf("expected ABC-123, got %s", issue.Identifier)
	}
}

func TestAssignIssue(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"data": map[string]any{
			"issueUpdate": map[string]any{
				"success": true,
				"issue": map[string]any{
					"id":       "issue-1",
					"assignee": map[string]any{"id": "user-1", "email": "dev@example.com"},
				},
			},
		},
	})
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	c.userID = "user-1"

	err := c.AssignIssue(context.Background(), "issue-1", "dev@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssignIssue_NoIdentity(t *testing.T) {
	c := NewLinearClient(Config{APIKey: "key"})
	// userID not set.
	err := c.AssignIssue(context.Background(), "issue-1", "dev@example.com")
	if err == nil {
		t.Fatal("expected error for missing identity")
	}
}

func TestUnassignIssue(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"data": map[string]any{
			"issueUpdate": map[string]any{
				"success": true,
				"issue":   map[string]any{"id": "issue-1"},
			},
		},
	})
	defer srv.Close()

	c := NewLinearClient(Config{Endpoint: srv.URL, APIKey: "key"})
	err := c.UnassignIssue(context.Background(), "issue-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
