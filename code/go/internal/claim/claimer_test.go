package claim

import (
	"context"
	"errors"
	"testing"

	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/tracker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements tracker.Client for testing.
type mockClient struct {
	assignErr   error
	fetchIssue  *domain.Issue
	fetchErr    error
	unassignErr error

	assignCalled   bool
	unassignCalled bool
}

func (m *mockClient) AssignIssue(_ context.Context, _, _ string) error {
	m.assignCalled = true
	return m.assignErr
}

func (m *mockClient) FetchIssue(_ context.Context, _ string) (*domain.Issue, error) {
	return m.fetchIssue, m.fetchErr
}

func (m *mockClient) UnassignIssue(_ context.Context, _ string) error {
	m.unassignCalled = true
	return m.unassignErr
}

func (m *mockClient) FetchCandidateIssues(_ context.Context) ([]domain.Issue, error) {
	return nil, nil
}

func (m *mockClient) FetchIssueStatesByIDs(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}

func (m *mockClient) FetchIssuesByStates(_ context.Context, _ []string) ([]domain.Issue, error) {
	return nil, nil
}

func TestClaim_Success(t *testing.T) {
	mock := &mockClient{
		fetchIssue: &domain.Issue{ID: "issue-1", Assignee: "user@example.com"},
	}

	err := Claim(context.Background(), mock, "issue-1", "user@example.com")

	require.NoError(t, err)
	assert.True(t, mock.assignCalled)
	assert.False(t, mock.unassignCalled)
}

func TestClaim_AssignError(t *testing.T) {
	mock := &mockClient{
		assignErr: errors.New("network error"),
	}

	err := Claim(context.Background(), mock, "issue-1", "user@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "claim assign")
	assert.Contains(t, err.Error(), "network error")
	// Should not proceed to fetch or unassign.
	assert.False(t, mock.unassignCalled)
}

func TestClaim_VerifyError(t *testing.T) {
	mock := &mockClient{
		fetchErr: errors.New("fetch failed"),
	}

	err := Claim(context.Background(), mock, "issue-1", "user@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "claim verify")
	assert.True(t, mock.assignCalled)
	assert.False(t, mock.unassignCalled)
}

func TestClaim_RaceLost(t *testing.T) {
	mock := &mockClient{
		fetchIssue: &domain.Issue{ID: "issue-1", Assignee: "other@example.com"},
	}

	err := Claim(context.Background(), mock, "issue-1", "user@example.com")

	require.Error(t, err)
	var conflict *tracker.ClaimConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, "issue-1", conflict.IssueID)
	assert.Equal(t, "user@example.com", conflict.ExpectedUser)
	assert.Equal(t, "other@example.com", conflict.ActualAssignee)
	assert.True(t, mock.unassignCalled, "should unassign on race loss")
}

func TestClaim_RaceLost_UnassignErrorIgnored(t *testing.T) {
	mock := &mockClient{
		fetchIssue:  &domain.Issue{ID: "issue-1", Assignee: "other@example.com"},
		unassignErr: errors.New("unassign failed"),
	}

	err := Claim(context.Background(), mock, "issue-1", "user@example.com")

	// Should still return ClaimConflictError even if unassign fails.
	var conflict *tracker.ClaimConflictError
	require.True(t, errors.As(err, &conflict))
}

func TestRelease_Success(t *testing.T) {
	mock := &mockClient{}

	err := Release(context.Background(), mock, "issue-1")

	require.NoError(t, err)
	assert.True(t, mock.unassignCalled)
}

func TestRelease_Error(t *testing.T) {
	mock := &mockClient{
		unassignErr: errors.New("api error"),
	}

	err := Release(context.Background(), mock, "issue-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "release")
	assert.Contains(t, err.Error(), "api error")
}
