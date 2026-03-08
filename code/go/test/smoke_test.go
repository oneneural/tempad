//go:build smoke

package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/tracker/linear"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Smoke tests require real Linear credentials:
//   LINEAR_API_KEY=... LINEAR_TEST_PROJECT_SLUG=... go test -tags smoke ./test/...

func TestSmoke_FetchCandidates(t *testing.T) {
	apiKey := os.Getenv("LINEAR_API_KEY")
	slug := os.Getenv("LINEAR_TEST_PROJECT_SLUG")
	if apiKey == "" || slug == "" {
		t.Skip("LINEAR_API_KEY and LINEAR_TEST_PROJECT_SLUG required")
	}

	client := linear.NewLinearClient(linear.Config{
		APIKey:         apiKey,
		ProjectSlug:    slug,
		ActiveStates:   []string{"Todo", "In Progress"},
		TerminalStates: []string{"Done", "Cancelled"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issues, err := client.FetchCandidateIssues(ctx)
	require.NoError(t, err)

	t.Logf("fetched %d candidate issues", len(issues))
	for _, iss := range issues {
		assert.NotEmpty(t, iss.ID)
		assert.NotEmpty(t, iss.Identifier)
		assert.NotEmpty(t, iss.Title)
		assert.NotEmpty(t, iss.State)
	}
}

func TestSmoke_ClaimAndRelease(t *testing.T) {
	apiKey := os.Getenv("LINEAR_API_KEY")
	slug := os.Getenv("LINEAR_TEST_PROJECT_SLUG")
	identity := os.Getenv("LINEAR_TEST_IDENTITY")
	if apiKey == "" || slug == "" || identity == "" {
		t.Skip("LINEAR_API_KEY, LINEAR_TEST_PROJECT_SLUG, and LINEAR_TEST_IDENTITY required")
	}

	client := linear.NewLinearClient(linear.Config{
		APIKey:         apiKey,
		ProjectSlug:    slug,
		Identity:       identity,
		ActiveStates:   []string{"Todo"},
		TerminalStates: []string{"Done"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch candidates.
	issues, err := client.FetchCandidateIssues(ctx)
	require.NoError(t, err)
	if len(issues) == 0 {
		t.Skip("no candidate issues found for claim/release test")
	}

	// Use the first unassigned issue.
	var target *struct {
		id string
	}
	for _, iss := range issues {
		if iss.Assignee == "" {
			target = &struct{ id string }{id: iss.ID}
			break
		}
	}
	if target == nil {
		t.Skip("no unassigned issue found")
	}

	// Assign.
	err = client.AssignIssue(ctx, target.id, identity)
	require.NoError(t, err)

	// Cleanup: always unassign.
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		client.UnassignIssue(cleanCtx, target.id)
	})

	// Verify assignment.
	issue, err := client.FetchIssue(ctx, target.id)
	require.NoError(t, err)
	assert.Equal(t, identity, issue.Assignee)

	// Small delay to avoid rate limiting.
	time.Sleep(500 * time.Millisecond)

	// Unassign.
	err = client.UnassignIssue(ctx, target.id)
	require.NoError(t, err)
}
