package test

import (
	"context"
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/logging"
	"github.com/oneneural/tempad/internal/orchestrator"
	"github.com/oneneural/tempad/internal/server"
	"github.com/oneneural/tempad/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- E2E test with mock tracker + mock launcher + real workspace + real server ---

func TestE2E_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "E2E-1", Title: "E2E Task 1", State: "Todo", CreatedAt: &now},
		{ID: "id-2", Identifier: "E2E-2", Title: "E2E Task 2", State: "Todo", CreatedAt: &now},
	}

	tracker := newE2ETracker(issues)
	launcher := newE2ELauncher(0, 50*time.Millisecond)

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	logger := logging.Setup(logging.Config{Level: "debug", Mode: "daemon"})

	cfg := &config.ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "test-key",
		TrackerProjectSlug: "E2E",
		TrackerIdentity:    "test@example.com",
		PollIntervalMs:     100,
		MaxConcurrent:      2,
		MaxRetries:         0,
		TerminalStates:     []string{"Done"},
		AgentCommand:       "echo done",
		StallTimeoutMs:     0,
	}

	orch := orchestrator.New(cfg, tracker, ws, logger, nil)
	orch.SetLauncher(launcher)

	// Start HTTP server on ephemeral port.
	srv, err := server.New(0, orch, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go srv.Serve(ctx)

	// Run orchestrator.
	err = orch.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify: issues were claimed.
	assert.GreaterOrEqual(t, tracker.assignCalls.Load(), int32(2),
		"both issues should have been claimed")

	// Verify: agents were launched.
	assert.GreaterOrEqual(t, launcher.launches.Load(), int32(2),
		"agents should have been launched for both issues")
}

func TestE2E_TerminalStateCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "E2E-T1", Title: "Task", State: "Todo", CreatedAt: &now},
	}

	tracker := newE2ETracker(issues)
	launcher := newE2ELauncher(0, 5*time.Second) // Long running.

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	logger := logging.Setup(logging.Config{Level: "error", Mode: "daemon"})

	cfg := &config.ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "test-key",
		TrackerProjectSlug: "E2E",
		TrackerIdentity:    "test@example.com",
		PollIntervalMs:     100,
		MaxConcurrent:      1,
		MaxRetries:         0,
		TerminalStates:     []string{"Done"},
		AgentCommand:       "sleep 60",
		StallTimeoutMs:     0,
	}

	orch := orchestrator.New(cfg, tracker, ws, logger, nil)
	orch.SetLauncher(launcher)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Move issue to terminal state after dispatch.
	go func() {
		time.Sleep(500 * time.Millisecond)
		tracker.setIssueState("id-1", "Done")
	}()

	err = orch.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Reconciliation should have detected the terminal state.
	assert.GreaterOrEqual(t, tracker.statesFetchCalls.Load(), int32(1))
}

func TestE2E_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "E2E-S1", Title: "Task", State: "Todo", CreatedAt: &now},
	}

	tracker := newE2ETracker(issues)
	launcher := newE2ELauncher(0, 10*time.Second) // Very long running.

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	logger := logging.Setup(logging.Config{Level: "error", Mode: "daemon"})

	cfg := &config.ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "test-key",
		TrackerProjectSlug: "E2E",
		TrackerIdentity:    "test@example.com",
		PollIntervalMs:     100,
		MaxConcurrent:      1,
		MaxRetries:         0,
		TerminalStates:     []string{"Done"},
		AgentCommand:       "sleep 60",
		StallTimeoutMs:     0,
	}

	orch := orchestrator.New(cfg, tracker, ws, logger, nil)
	orch.SetLauncher(launcher)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- orch.Run(ctx)
	}()

	// Wait for agent to start, then cancel.
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Should shut down within 5s.
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown took too long")
	}

	// Claims should have been released.
	assert.GreaterOrEqual(t, tracker.unassignCalls.Load(), int32(1),
		"claims should be released on shutdown")
}
