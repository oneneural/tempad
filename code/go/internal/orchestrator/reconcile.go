package orchestrator

import (
	"context"
	"time"

	"github.com/oneneural/tempad/internal/domain"
)

// reconcile checks running issues for stalls and terminal state changes.
func (o *Orchestrator) reconcile(ctx context.Context) {
	if len(o.state.Running) == 0 {
		return
	}

	// Part A: Stall detection.
	o.detectStalls(ctx)

	// Part B: Tracker state refresh.
	o.refreshTrackerStates(ctx)
}

// detectStalls checks if any running agent has not produced output recently.
func (o *Orchestrator) detectStalls(ctx context.Context) {
	stallThreshold := time.Duration(o.cfg.StallTimeoutMs) * time.Millisecond
	if stallThreshold <= 0 {
		return
	}

	now := time.Now().UnixNano()
	for issueID, run := range o.state.Running {
		lastOutput, ok := o.lastOutput[issueID]
		if !ok {
			continue
		}

		lastAt := lastOutput.Load()
		elapsed := time.Duration(now - lastAt)
		if elapsed > stallThreshold {
			o.logger.Warn("agent stalled, canceling",
				"issue", run.IssueIdentifier,
				"elapsed", elapsed,
				"threshold", stallThreshold,
			)
			run.Status = "stalled"
			if cancel, ok := o.workerCancels[issueID]; ok {
				cancel()
			}
		}
	}
}

// refreshTrackerStates fetches current states for running issues from the
// tracker and handles terminal state changes.
func (o *Orchestrator) refreshTrackerStates(ctx context.Context) {
	var runningIDs []string
	for id := range o.state.Running {
		runningIDs = append(runningIDs, id)
	}

	if len(runningIDs) == 0 {
		return
	}

	states, err := o.tracker.FetchIssueStatesByIDs(ctx, runningIDs)
	if err != nil {
		// API unreachable — keep workers running (safe default).
		o.logger.Warn("state refresh failed, keeping workers", "error", err)
		return
	}

	terminalStates := domain.NormalizeStates(o.cfg.TerminalStates)

	for issueID, state := range states {
		run, running := o.state.Running[issueID]
		if !running {
			continue
		}

		normalized := domain.NormalizeState(state)
		if terminalStates[normalized] {
			// Terminal state — cancel worker, clean workspace.
			o.logger.Info("issue terminal, canceling worker",
				"issue", run.IssueIdentifier,
				"state", state,
			)
			run.Status = "canceled"
			if cancel, ok := o.workerCancels[issueID]; ok {
				cancel()
			}

			// Clean workspace.
			if cleanErr := o.ws.CleanForIssue(run.IssueIdentifier); cleanErr != nil {
				o.logger.Warn("workspace cleanup failed",
					"issue", run.IssueIdentifier,
					"error", cleanErr,
				)
			}

			// Remove claim.
			delete(o.state.Claimed, issueID)
		}
	}
}
