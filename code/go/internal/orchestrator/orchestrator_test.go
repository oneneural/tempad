package orchestrator

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testConfig() *config.ServiceConfig {
	return &config.ServiceConfig{
		PollIntervalMs: 1000,
		MaxConcurrent:  3,
	}
}

func TestNew(t *testing.T) {
	cfg := testConfig()
	o := New(cfg, nil, nil, testLogger())

	require.NotNil(t, o)
	assert.Equal(t, 1000, o.state.PollIntervalMs)
	assert.Equal(t, 3, o.state.MaxConcurrentAgents)
	assert.NotNil(t, o.workerResults)
	assert.NotNil(t, o.retryTimers)
	assert.NotNil(t, o.configReload)
	assert.Equal(t, 0, o.state.RunningCount())
	assert.Equal(t, 3, o.state.AvailableSlots())
}

func TestRun_ContextCancel(t *testing.T) {
	cfg := testConfig()
	o := New(cfg, nil, nil, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestReloadConfig(t *testing.T) {
	cfg := testConfig()
	o := New(cfg, nil, nil, testLogger())

	newCfg := &config.ServiceConfig{PollIntervalMs: 5000, MaxConcurrent: 10}
	o.ReloadConfig(newCfg)

	select {
	case received := <-o.configReload:
		assert.Equal(t, 5000, received.PollIntervalMs)
	default:
		t.Fatal("expected config on reload channel")
	}
}

func TestState(t *testing.T) {
	o := New(testConfig(), nil, nil, testLogger())
	state := o.State()
	assert.NotNil(t, state)
	assert.Equal(t, 3, state.MaxConcurrentAgents)
}

func TestApplyNewConfig(t *testing.T) {
	o := New(testConfig(), nil, nil, testLogger())

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	newCfg := &config.ServiceConfig{PollIntervalMs: 5000, MaxConcurrent: 10}
	o.applyNewConfig(newCfg, ticker)

	assert.Equal(t, 5000, o.state.PollIntervalMs)
	assert.Equal(t, 10, o.state.MaxConcurrentAgents)
	assert.Equal(t, newCfg, o.cfg)
}

func TestHandleRetry_ContextCanceled(t *testing.T) {
	o := New(testConfig(), nil, nil, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already canceled.

	// Should not panic or do anything.
	o.handleRetry(ctx, RetrySignal{IssueID: "id-1", Identifier: "PROJ-1", Attempt: 1})
}

func TestWorkerResultChannel(t *testing.T) {
	o := New(testConfig(), nil, nil, testLogger())

	// Send a worker result.
	o.workerResults <- WorkerResult{
		IssueID:    "id-1",
		Identifier: "PROJ-1",
		ExitCode:   0,
		Duration:   5 * time.Second,
	}

	// Should receive it.
	select {
	case r := <-o.workerResults:
		assert.Equal(t, "PROJ-1", r.Identifier)
		assert.Equal(t, 0, r.ExitCode)
	default:
		t.Fatal("expected result on channel")
	}
}
