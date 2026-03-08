package orchestrator

import (
	"context"
	"sort"
	"time"

	"github.com/oneneural/tempad/internal/claim"
	"github.com/oneneural/tempad/internal/domain"
)

// selectCandidates filters and sorts issues for dispatch eligibility.
// An issue is eligible only if:
//   - It has id, identifier, title, and state.
//   - Its state is in active_states and not in terminal_states.
//   - It is unassigned, or assigned to the current user (for resumption).
//   - It is not already in running, claimed, or retry.
//   - Blocker rule for Todo state passes (no non-terminal blockers).
//
// Sorting: priority asc (null last) → created_at oldest → identifier.
func (o *Orchestrator) selectCandidates(issues []domain.Issue) []domain.Issue {
	terminalStates := domain.NormalizeStates(o.cfg.TerminalStates)
	identity := o.cfg.TrackerIdentity

	var eligible []domain.Issue
	for _, issue := range issues {
		if !hasRequiredFields(issue) {
			continue
		}
		if issue.Assignee != "" && issue.Assignee != identity {
			continue
		}
		if o.state.IsClaimedOrRunning(issue.ID) {
			continue
		}
		if _, retrying := o.state.RetryAttempts[issue.ID]; retrying {
			continue
		}
		// Blocker rule: Todo issues with non-terminal blockers are ineligible.
		if domain.NormalizeState(issue.State) == "todo" && issue.HasNonTerminalBlockers(terminalStates) {
			continue
		}
		eligible = append(eligible, issue)
	}

	sortForDispatch(eligible)
	return eligible
}

// hasRequiredFields checks that the issue has all fields needed for dispatch.
func hasRequiredFields(issue domain.Issue) bool {
	return issue.ID != "" && issue.Identifier != "" && issue.Title != "" && issue.State != ""
}

// sortForDispatch sorts issues by: priority asc (null last) → created_at oldest → identifier.
func sortForDispatch(issues []domain.Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]

		ap := priorityVal(a.Priority)
		bp := priorityVal(b.Priority)
		if ap != bp {
			return ap < bp
		}

		if a.CreatedAt != nil && b.CreatedAt != nil {
			if !a.CreatedAt.Equal(*b.CreatedAt) {
				return a.CreatedAt.Before(*b.CreatedAt)
			}
		} else if a.CreatedAt != nil {
			return true
		} else if b.CreatedAt != nil {
			return false
		}

		return a.Identifier < b.Identifier
	})
}

func priorityVal(p *int) int {
	if p == nil {
		return 999
	}
	return *p
}

// stateSlotAvailable checks if there's a per-state concurrency slot available
// for the given issue state. If no per-state limit is configured, returns true
// (fall back to global limit). Invalid entries (non-positive) are ignored.
// dispatch iterates eligible candidates and claims + spawns workers.
func (o *Orchestrator) dispatch(ctx context.Context, candidates []domain.Issue) {
	for _, issue := range candidates {
		if o.state.AvailableSlots() <= 0 {
			break
		}
		if !o.stateSlotAvailable(issue.State) {
			continue
		}

		// Claim the issue.
		if err := claim.Claim(ctx, o.tracker, issue.ID, o.cfg.TrackerIdentity); err != nil {
			o.logger.Warn("claim failed, skipping",
				"issue", issue.Identifier,
				"error", err,
			)
			continue
		}

		o.state.Claimed[issue.ID] = true

		// Spawn worker goroutine.
		attempt := 0
		run := &domain.RunAttempt{
			IssueID:         issue.ID,
			IssueIdentifier: issue.Identifier,
			Attempt:         &attempt,
			StartedAt:       time.Now(),
			Status:          "running",
		}
		o.state.Running[issue.ID] = run

		go o.runWorker(ctx, issue, attempt)

		o.logger.Info("dispatched",
			"issue", issue.Identifier,
			"slots_remaining", o.state.AvailableSlots(),
		)
	}
}

func (o *Orchestrator) stateSlotAvailable(state string) bool {
	if len(o.cfg.MaxConcurrentByState) == 0 {
		return true
	}

	normalized := domain.NormalizeState(state)
	limit, exists := o.cfg.MaxConcurrentByState[normalized]
	if !exists {
		return true // No per-state limit.
	}
	if limit <= 0 {
		return true // Invalid entry, ignore.
	}

	// Count running issues in this state.
	count := 0
	for _, run := range o.state.Running {
		if domain.NormalizeState(run.Status) == normalized {
			count++
		}
	}

	return count < limit
}
