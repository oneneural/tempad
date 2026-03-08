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
// is canceled. Placeholder — the select loop is implemented in T-P501.
func (o *Orchestrator) Run(ctx context.Context) error {
	o.logger.Info("orchestrator starting",
		"poll_interval_ms", o.cfg.PollIntervalMs,
		"max_concurrent", o.cfg.MaxConcurrent,
	)

	// Placeholder — T-P501 implements the select loop.
	<-ctx.Done()

	o.shutdown()
	return ctx.Err()
}

// shutdown cleans up on exit: stop timers, drain channels.
func (o *Orchestrator) shutdown() {
	o.logger.Info("orchestrator shutting down")

	// Cancel all retry timers.
	for id, timer := range o.activeTimers {
		timer.Stop()
		delete(o.activeTimers, id)
	}
}
