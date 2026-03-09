// Package orchestrator implements the daemon-mode poll-dispatch-reconcile loop.
package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"github.com/oneneural/tempad/internal/agent"
	"github.com/oneneural/tempad/internal/claim"
	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/notify"
	"github.com/oneneural/tempad/internal/prompt"
	"github.com/oneneural/tempad/internal/tracker"
	"github.com/oneneural/tempad/internal/workspace"
)

// WorkerResult is sent by a worker goroutine when an agent finishes.
type WorkerResult struct {
	IssueID    string
	Identifier string
	ExitCode   int
	Duration   time.Duration
	Attempt    int  // 0-based attempt index
	Err        error
}

// RetrySignal is sent when a retry timer fires.
type RetrySignal struct {
	IssueID    string
	Identifier string
	Attempt    int
}

// Orchestrator is the daemon-mode orchestrator. It owns all mutable scheduling
// state. No other goroutine reads or writes the state — communication is via
// channels. See Architecture doc Section 5.3.
type Orchestrator struct {
	cfg      *config.ServiceConfig
	tracker  tracker.Client
	ws       *workspace.Manager
	launcher agent.Launcher
	builder  *prompt.Builder
	state    *domain.OrchestratorState
	logger   *slog.Logger
	notifier *notify.Notifier

	// Channels — all buffered to prevent goroutine leaks.
	workerResults chan WorkerResult
	retryTimers   chan RetrySignal
	configReload  chan *config.ServiceConfig

	// Retry timer handles for cancellation.
	activeTimers map[string]*time.Timer // issue_id → timer

	// Stall detection: issue_id → last output timestamp (Unix nanos).
	lastOutput map[string]*atomic.Int64

	// Worker cancel functions for killing agents.
	workerCancels map[string]func() // issue_id → cancel
}

// New creates a new Orchestrator with initialized state and channels.
func New(cfg *config.ServiceConfig, client tracker.Client, ws *workspace.Manager, logger *slog.Logger, notifier *notify.Notifier) *Orchestrator {
	if notifier == nil {
		notifier = notify.Noop()
	}
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	return &Orchestrator{
		cfg:      cfg,
		tracker:  client,
		ws:       ws,
		launcher: agent.NewSubprocessLauncher(),
		builder:  prompt.NewBuilder(),
		state:    domain.NewOrchestratorState(cfg.PollIntervalMs, maxConcurrent),
		logger:   logger,

		notifier: notifier,

		workerResults: make(chan WorkerResult, maxConcurrent),
		retryTimers:   make(chan RetrySignal, maxConcurrent),
		configReload:  make(chan *config.ServiceConfig, 1),

		activeTimers:  make(map[string]*time.Timer),
		lastOutput:    make(map[string]*atomic.Int64),
		workerCancels: make(map[string]func()),
	}
}

// State returns the orchestrator's state for external reads (HTTP API).
func (o *Orchestrator) State() *domain.OrchestratorState {
	return o.state
}

// SetLauncher overrides the agent launcher (used in tests).
func (o *Orchestrator) SetLauncher(l agent.Launcher) {
	o.launcher = l
}

// ReloadConfig sends a new config to the orchestrator for application on the
// next tick. Non-blocking — drops the config if the channel is full.
func (o *Orchestrator) ReloadConfig(cfg *config.ServiceConfig) {
	select {
	case o.configReload <- cfg:
	default:
		o.logger.Warn("config reload channel full, skipping")
	}
}

// TriggerPoll forces an immediate poll cycle by reloading the current config,
// which resets the ticker. Used by the HTTP refresh endpoint.
func (o *Orchestrator) TriggerPoll() {
	o.ReloadConfig(o.cfg)
}

// Run starts the orchestrator's main select loop. It blocks until the context
// is canceled. The select loop handles:
//   - ctx.Done: graceful shutdown
//   - ticker: periodic poll → reconcile → dispatch
//   - workerResults: agent exit handling
//   - retryTimers: retry scheduling
//   - configReload: apply new config
func (o *Orchestrator) Run(ctx context.Context) error {
	o.logger.Info("orchestrator starting",
		"poll_interval_ms", o.cfg.PollIntervalMs,
		"max_concurrent", o.cfg.MaxConcurrent,
	)

	interval := time.Duration(o.cfg.PollIntervalMs) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Immediate first tick.
	o.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			o.shutdown()
			return ctx.Err()

		case <-ticker.C:
			o.tick(ctx)

		case result := <-o.workerResults:
			o.handleWorkerExit(ctx, result)

		case signal := <-o.retryTimers:
			o.handleRetry(ctx, signal)

		case cfg := <-o.configReload:
			o.applyNewConfig(cfg, ticker)
		}
	}
}

// tick runs one poll-reconcile-dispatch cycle.
func (o *Orchestrator) tick(ctx context.Context) {
	o.logger.Debug("tick",
		"running", o.state.RunningCount(),
		"slots", o.state.AvailableSlots(),
	)

	// Reconcile running issues.
	o.reconcile(ctx)

	// Validate dispatch config.
	if err := config.ValidateForStartup(o.cfg, "daemon"); err != nil {
		o.logger.Warn("dispatch config invalid, skipping", "error", err)
		return
	}

	// Fetch candidates.
	if o.state.AvailableSlots() <= 0 {
		return
	}
	issues, err := o.tracker.FetchCandidateIssues(ctx)
	if err != nil {
		o.logger.Error("fetch candidates failed", "error", err)
		return
	}

	o.logger.Debug("fetched candidates", "total", len(issues))
	for _, iss := range issues {
		o.logger.Debug("candidate", "identifier", iss.Identifier, "title", iss.Title, "state", iss.State, "assignee", iss.Assignee)
	}

	candidates := o.selectCandidates(issues)
	o.dispatch(ctx, candidates)
}

// handleWorkerExit processes a worker result when an agent finishes.
func (o *Orchestrator) handleWorkerExit(ctx context.Context, result WorkerResult) {
	o.logger.Info("worker exit",
		"issue", result.Identifier,
		"exit_code", result.ExitCode,
		"duration", result.Duration,
		"attempt", result.Attempt,
	)

	// Remove from running.
	delete(o.state.Running, result.IssueID)
	delete(o.lastOutput, result.IssueID)
	delete(o.workerCancels, result.IssueID)

	// Update totals.
	o.state.AgentTotals.TotalRuntimeSeconds += result.Duration.Seconds()

	if result.ExitCode == 0 {
		// Normal exit — mark completed, schedule continuation retry (1s).
		o.state.Completed[result.IssueID] = true
		o.notifier.Send(notify.EventAgentCompleted, "TEMPAD: Task Completed",
			fmt.Sprintf("%s (exit 0, %s)", result.Identifier, result.Duration.Truncate(time.Second)))
		o.scheduleRetry(ctx, result.IssueID, result.Identifier, result.Attempt, true, "")
	} else {
		// Failure — schedule exponential backoff retry.
		errMsg := fmt.Sprintf("exit code %d", result.ExitCode)
		if result.Err != nil {
			errMsg = result.Err.Error()
		}
		o.notifier.Send(notify.EventAgentFailed, "TEMPAD: Task Failed",
			fmt.Sprintf("%s (exit %d, attempt %d)", result.Identifier, result.ExitCode, result.Attempt+1))
		o.scheduleRetry(ctx, result.IssueID, result.Identifier, result.Attempt+1, false, errMsg)
	}
}

// handleRetry processes a retry signal when a timer fires.
func (o *Orchestrator) handleRetry(ctx context.Context, signal RetrySignal) {
	if ctx.Err() != nil {
		return
	}

	// Remove from retry queue.
	delete(o.state.RetryAttempts, signal.IssueID)
	delete(o.activeTimers, signal.IssueID)

	o.logger.Info("retry firing",
		"issue", signal.Identifier,
		"attempt", signal.Attempt,
	)

	// Check if issue is still active.
	issue, err := o.tracker.FetchIssue(ctx, signal.IssueID)
	if err != nil || issue == nil {
		o.logger.Warn("retry: issue not found, releasing claim",
			"issue", signal.Identifier,
		)
		_ = o.tracker.UnassignIssue(ctx, signal.IssueID)
		delete(o.state.Claimed, signal.IssueID)
		return
	}

	// Check max retries (failure retries only).
	entry, hasEntry := o.state.RetryAttempts[signal.IssueID]
	if hasEntry && !entry.IsContinuation && signal.Attempt > o.cfg.MaxRetries {
		o.logger.Info("max retries exhausted, releasing claim",
			"issue", signal.Identifier,
			"attempt", signal.Attempt,
		)
		o.notifier.Send(notify.EventRetriesExhausted, "TEMPAD: Retries Exhausted",
			fmt.Sprintf("%s: claim released after %d attempts", signal.Identifier, signal.Attempt))
		_ = claim.Release(ctx, o.tracker, signal.IssueID)
		delete(o.state.Claimed, signal.IssueID)
		return
	}

	// Check slots.
	if o.state.AvailableSlots() <= 0 {
		o.logger.Debug("retry: no slots, requeueing",
			"issue", signal.Identifier,
		)
		// Requeue with same attempt.
		isContinuation := hasEntry && entry.IsContinuation
		o.scheduleRetry(ctx, signal.IssueID, signal.Identifier, signal.Attempt, isContinuation, "no slots")
		return
	}

	// Dispatch.
	run := &domain.RunAttempt{
		IssueID:         signal.IssueID,
		IssueIdentifier: signal.Identifier,
		Attempt:         &signal.Attempt,
		StartedAt:       time.Now(),
		Status:          "running",
	}
	o.state.Running[signal.IssueID] = run

	// Pre-allocate cancel and output monitor on orchestrator goroutine.
	workerCtx, workerCancel := context.WithCancel(ctx)
	o.workerCancels[signal.IssueID] = workerCancel

	lastOutput := &atomic.Int64{}
	lastOutput.Store(time.Now().UnixNano())
	o.lastOutput[signal.IssueID] = lastOutput

	go o.runWorker(workerCtx, *issue, signal.Attempt, lastOutput)
}

// scheduleRetry schedules a retry timer for an issue.
func (o *Orchestrator) scheduleRetry(ctx context.Context, issueID, identifier string, attempt int, isContinuation bool, errMsg string) {
	// Cancel any existing timer.
	if timer, ok := o.activeTimers[issueID]; ok {
		timer.Stop()
		delete(o.activeTimers, issueID)
	}

	// Check max retries for failure retries.
	if !isContinuation && attempt > o.cfg.MaxRetries {
		o.logger.Info("max retries exhausted, releasing claim",
			"issue", identifier,
			"attempt", attempt,
		)
		o.notifier.Send(notify.EventRetriesExhausted, "TEMPAD: Retries Exhausted",
			fmt.Sprintf("%s: claim released after %d attempts", identifier, attempt))
		_ = claim.Release(ctx, o.tracker, issueID)
		delete(o.state.Claimed, issueID)
		return
	}

	// Compute delay.
	var delay time.Duration
	if isContinuation {
		delay = 1 * time.Second
	} else {
		// Exponential backoff: min(10000 * 2^(attempt-1), max_retry_backoff_ms).
		backoffMs := 10000.0 * math.Pow(2, float64(attempt-1))
		maxMs := float64(o.cfg.MaxRetryBackoffMs)
		if maxMs <= 0 {
			maxMs = 300000
		}
		delay = time.Duration(math.Min(backoffMs, maxMs)) * time.Millisecond
	}

	// Store retry entry.
	o.state.RetryAttempts[issueID] = &domain.RetryEntry{
		IssueID:        issueID,
		Identifier:     identifier,
		Attempt:        attempt,
		DueAtMs:        time.Now().Add(delay).UnixMilli(),
		Error:          errMsg,
		IsContinuation: isContinuation,
	}

	// Schedule timer.
	timer := time.AfterFunc(delay, func() {
		if ctx.Err() != nil {
			return
		}
		o.retryTimers <- RetrySignal{
			IssueID:    issueID,
			Identifier: identifier,
			Attempt:    attempt,
		}
	})
	o.activeTimers[issueID] = timer

	o.logger.Info("retry scheduled",
		"issue", identifier,
		"attempt", attempt,
		"delay", delay,
		"continuation", isContinuation,
	)
}

// applyNewConfig applies a reloaded config to the orchestrator.
// In-flight agents are not restarted — changes apply to the next dispatch cycle.
func (o *Orchestrator) applyNewConfig(cfg *config.ServiceConfig, ticker *time.Ticker) {
	old := o.cfg

	// Log changed fields for operational debugging.
	if old.PollIntervalMs != cfg.PollIntervalMs {
		o.logger.Info("config changed: poll_interval_ms",
			"old", old.PollIntervalMs, "new", cfg.PollIntervalMs)
	}
	if old.MaxConcurrent != cfg.MaxConcurrent {
		o.logger.Info("config changed: max_concurrent",
			"old", old.MaxConcurrent, "new", cfg.MaxConcurrent)
	}
	if old.MaxRetries != cfg.MaxRetries {
		o.logger.Info("config changed: max_retries",
			"old", old.MaxRetries, "new", cfg.MaxRetries)
	}
	if old.MaxRetryBackoffMs != cfg.MaxRetryBackoffMs {
		o.logger.Info("config changed: max_retry_backoff_ms",
			"old", old.MaxRetryBackoffMs, "new", cfg.MaxRetryBackoffMs)
	}
	if old.StallTimeoutMs != cfg.StallTimeoutMs {
		o.logger.Info("config changed: stall_timeout_ms",
			"old", old.StallTimeoutMs, "new", cfg.StallTimeoutMs)
	}
	if old.AgentCommand != cfg.AgentCommand {
		o.logger.Info("config changed: agent_command",
			"old", old.AgentCommand, "new", cfg.AgentCommand)
	}

	o.cfg = cfg

	// Update state limits.
	o.state.PollIntervalMs = cfg.PollIntervalMs
	o.state.MaxConcurrentAgents = cfg.MaxConcurrent

	// Reset ticker interval.
	newInterval := time.Duration(cfg.PollIntervalMs) * time.Millisecond
	ticker.Reset(newInterval)
}

// shutdown cleans up on exit: cancel workers, stop timers, release claims.
func (o *Orchestrator) shutdown() {
	runningCount := o.state.RunningCount()
	o.logger.Info("orchestrator shutting down",
		"running", runningCount,
		"claimed", len(o.state.Claimed),
	)

	// Cancel all retry timers.
	for id, timer := range o.activeTimers {
		timer.Stop()
		delete(o.activeTimers, id)
	}

	// Cancel all running workers explicitly and wait for them to finish.
	for id, cancel := range o.workerCancels {
		cancel()
		delete(o.workerCancels, id)
	}

	// Drain remaining worker results so goroutines can exit cleanly.
	for i := 0; i < runningCount; i++ {
		select {
		case <-o.workerResults:
		case <-time.After(5 * time.Second):
			o.logger.Warn("timed out waiting for worker to finish")
		}
	}

	// Release all claims (best effort).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for issueID := range o.state.Claimed {
		if err := o.tracker.UnassignIssue(ctx, issueID); err != nil {
			o.logger.Warn("failed to release claim on shutdown",
				"issue", issueID,
				"error", err,
			)
		}
	}
}
