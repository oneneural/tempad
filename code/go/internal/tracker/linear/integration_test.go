//go:build integration

package linear

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests require:
//   LINEAR_API_KEY           — Linear API key
//   LINEAR_TEST_PROJECT_SLUG — project slug to query
//   LINEAR_TEST_IDENTITY     — email for identity resolution
//
// Run with: go test -tags=integration -race ./internal/tracker/linear/

func TestMain(m *testing.M) {
	if os.Getenv("LINEAR_API_KEY") == "" || os.Getenv("LINEAR_TEST_PROJECT_SLUG") == "" {
		// Skip all integration tests when env vars are missing.
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func newIntegrationClient(t *testing.T) *LinearClient {
	t.Helper()

	c := NewLinearClient(Config{
		APIKey:       os.Getenv("LINEAR_API_KEY"),
		ProjectSlug:  os.Getenv("LINEAR_TEST_PROJECT_SLUG"),
		Identity:     os.Getenv("LINEAR_TEST_IDENTITY"),
		ActiveStates: []string{"Todo", "In Progress"},
		TerminalStates: []string{"Done", "Canceled", "Cancelled", "Closed", "Duplicate"},
	})

	if c.identity != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := c.ResolveIdentity(ctx); err != nil {
			t.Fatalf("resolve identity: %v", err)
		}
	}

	return c
}

func TestIntegration_ResolveIdentity(t *testing.T) {
	c := newIntegrationClient(t)
	if c.userID == "" {
		t.Error("expected userID to be resolved")
	}
	t.Logf("Resolved identity %q → userID %q", c.identity, c.userID)
}

func TestIntegration_FetchCandidateIssues(t *testing.T) {
	c := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issues, err := c.FetchCandidateIssues(ctx)
	if err != nil {
		t.Fatalf("FetchCandidateIssues: %v", err)
	}

	t.Logf("Found %d candidate issues", len(issues))
	for _, issue := range issues {
		t.Logf("  %s: %s (state=%s, priority=%v)", issue.Identifier, issue.Title, issue.State, issue.Priority)
		if issue.ID == "" {
			t.Error("issue has empty ID")
		}
		if issue.Identifier == "" {
			t.Error("issue has empty Identifier")
		}
	}
}

func TestIntegration_FetchIssuesByStates(t *testing.T) {
	c := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issues, err := c.FetchIssuesByStates(ctx, c.terminalStates)
	if err != nil {
		t.Fatalf("FetchIssuesByStates: %v", err)
	}

	t.Logf("Found %d terminal issues", len(issues))
}

func TestIntegration_FetchSingleIssue(t *testing.T) {
	c := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First fetch candidates to get a real issue ID.
	issues, err := c.FetchCandidateIssues(ctx)
	if err != nil {
		t.Fatalf("FetchCandidateIssues: %v", err)
	}
	if len(issues) == 0 {
		t.Skip("no candidate issues available for single fetch test")
	}

	issue, err := c.FetchIssue(ctx, issues[0].ID)
	if err != nil {
		t.Fatalf("FetchIssue: %v", err)
	}

	if issue.ID != issues[0].ID {
		t.Errorf("expected ID=%s, got %s", issues[0].ID, issue.ID)
	}
	t.Logf("Fetched single issue: %s %s", issue.Identifier, issue.Title)
}
