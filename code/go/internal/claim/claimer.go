// Package claim implements the assignment-based distributed claiming mechanism.
// Used by both TUI and daemon modes for issue assignment with race detection.
package claim

import (
	"context"
	"fmt"

	"github.com/oneneural/tempad/internal/tracker"
)

// Claim assigns an issue to the given identity and verifies the assignment.
// If another user claimed the issue in the race window, it unassigns and returns
// a ClaimConflictError. This is stateless — all state is managed by the caller.
func Claim(ctx context.Context, client tracker.Client, issueID, identity string) error {
	if err := client.AssignIssue(ctx, issueID, identity); err != nil {
		return fmt.Errorf("claim assign: %w", err)
	}

	issue, err := client.FetchIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("claim verify: %w", err)
	}

	if issue.Assignee != identity {
		// Race lost — someone else claimed it between assign and verify.
		// Best-effort unassign to clean up our assignment attempt.
		_ = client.UnassignIssue(ctx, issueID)

		return &tracker.ClaimConflictError{
			IssueID:        issueID,
			ExpectedUser:   identity,
			ActualAssignee: issue.Assignee,
		}
	}

	return nil
}

// Release removes the assignment from an issue.
func Release(ctx context.Context, client tracker.Client, issueID string) error {
	if err := client.UnassignIssue(ctx, issueID); err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}
