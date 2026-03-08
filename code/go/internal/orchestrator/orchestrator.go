// Package orchestrator implements the daemon-mode poll-dispatch-reconcile loop.
package orchestrator

import (
	"context"
	"log/slog"
	"time"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
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
	cfg     *config.ServiceConfig
	tracker tracker.Client
	ws      *workspace.Manager
	state   *domain.OrchestratorState
	logger  *slog.Logger

	// Channels — all buffered to prevent goroutine leaks.
	workerResults chan WorkerResult
	retryTimers   chan RetrySignal
	configReload  chan *config.ServiceConfig

	// Retry timer handles for cancellation.
	activeTimers map[string]*time.Timer // issue_id → timer
}

// New creates a new Orchestrator with initialized state and channels.
func New(cfg *config.ServiceConfig, client tracker.Client, ws *workspace.Manager, logger *slog.Logger) *Orchestrator {
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	return &Orchestrator{
		cfg:     cfg,
		tracker: client,
		ws:      ws,
		state:   domain.NewOrchestratorState(cfg.PollIntervalMs, maxConcurrent),
		logger:  logger,

		workerResults: make(chan WorkerResult, maxConcurrent),
		retryTimers:   make(chan RetrySignal, maxConcurrent),
		configReload:  make(chan *config.ServiceConfig, 1),

		activeTimers: make(map[string]*time.Timer),
	}
}

// State returns the orchestrator's state for external reads (HTTP API).
func (o *Orchestrator) State() *domain.OrchestratorState {
	return o.state
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
// Placeholder — reconciliation (T-P511) and dispatch (T-P504) added later.
func (o *Orchestrator) tick(ctx context.Context) {
	o.logger.Debug("tick",
		"running", o.state.RunningCount(),
		"slots", o.state.AvailableSlots(),
	)
	// TODO(T-P511): reconcile running issues.
	// TODO(T-P504): dispatch eligible candidates.
}

// handleWorkerExit processes a worker result when an agent finishes.
// Placeholder — worker exit handling (T-P509) added later.
func (o *Orchestrator) handleWorkerExit(ctx context.Context, result WorkerResult) {
	o.logger.Info("worker exit",
		"issue", result.Identifier,
		"exit_code", result.ExitCode,
		"duration", result.Duration,
	)
	// TODO(T-P509): handle exit, schedule retry.
}

// handleRetry processes a retry signal when a timer fires.
// Placeholder — retry handling (T-P510) added later.
func (o *Orchestrator) handleRetry(ctx context.Context, signal RetrySignal) {
	if ctx.Err() != nil {
		return // Don't retry during shutdown.
	}
	o.logger.Info("retry signal",
		"issue", signal.Identifier,
		"attempt", signal.Attempt,
	)
	// TODO(T-P510): check issue state, dispatch if eligible.
}

// applyNewConfig applies a reloaded config to the orchestrator.
func (o *Orchestrator) applyNewConfig(cfg *config.ServiceConfig, ticker *time.Ticker) {
	o.logger.Info("applying new config",
		"poll_interval_ms", cfg.PollIntervalMs,
		"max_concurrent", cfg.MaxConcurrent,
	)
	o.cfg = cfg

	// Update state limits.
	o.state.PollIntervalMs = cfg.PollIntervalMs
	o.state.MaxConcurrentAgents = cfg.MaxConcurrent

	// Reset ticker interval.
	newInterval := time.Duration(cfg.PollIntervalMs) * time.Millisecond
	ticker.Reset(newInterval)
}

// shutdown cleans up on exit: stop timers, release claims.
func (o *Orchestrator) shutdown() {
	o.logger.Info("orchestrator shutting down",
		"running", o.state.RunningCount(),
		"claimed", len(o.state.Claimed),
	)

	// Cancel all retry timers.
	for id, timer := range o.activeTimers {
		timer.Stop()
		delete(o.activeTimers, id)
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
