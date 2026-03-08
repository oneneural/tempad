// Package tracker defines the issue tracker abstraction.
// See Spec Section 13.1 and Architecture doc Section 4.1.
package tracker

import (
	"context"

	"github.com/oneneural/tempad/internal/domain"
)

// Client is the issue tracker interface. Linear is the first implementation.
// Future adapters (Jira, GitHub Issues) add sibling packages.
type Client interface {
	// FetchCandidateIssues returns unassigned issues in active states
	// for the configured project. Used for dispatch candidate selection.
	FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error)

	// FetchIssueStatesByIDs returns current states for the given issue IDs.
	// Used for reconciliation (daemon mode).
	FetchIssueStatesByIDs(ctx context.Context, ids []string) (map[string]string, error)

	// FetchIssuesByStates returns issues in the given states.
	// Used for startup cleanup of terminal workspaces.
	FetchIssuesByStates(ctx context.Context, states []string) ([]domain.Issue, error)

	// FetchIssue returns a single issue by ID.
	// Used for claim verification.
	FetchIssue(ctx context.Context, id string) (*domain.Issue, error)

	// AssignIssue assigns the issue to the given identity (claim).
	AssignIssue(ctx context.Context, issueID string, identity string) error

	// UnassignIssue removes assignment from the issue (release claim).
	UnassignIssue(ctx context.Context, issueID string) error
}
