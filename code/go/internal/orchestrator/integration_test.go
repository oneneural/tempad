package orchestrator

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/agent"
	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Tracker ---

type mockTracker struct {
	mu               sync.Mutex
	issues           []domain.Issue
	states           map[string]string // id → state
	assigned         map[string]string // id → identity
	fetchCalls       atomic.Int32
	assignCalls      atomic.Int32
	unassignCalls    atomic.Int32
	statesFetchCalls atomic.Int32
}

func newMockTracker(issues []domain.Issue) *mockTracker {
	states := make(map[string]string)
	for _, iss := range issues {
		states[iss.ID] = iss.State
	}
	return &mockTracker{
		issues:   issues,
		states:   states,
		assigned: make(map[string]string),
	}
}

func (m *mockTracker) FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error) {
	m.fetchCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []domain.Issue
	for _, iss := range m.issues {
		cp := iss
		cp.Assignee = m.assigned[iss.ID]
		if st, ok := m.states[iss.ID]; ok {
			cp.State = st
		}
		result = append(result, cp)
	}
	return result, nil
}

func (m *mockTracker) FetchIssueStatesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	m.statesFetchCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]string)
	for _, id := range ids {
		if st, ok := m.states[id]; ok {
			result[id] = st
		}
	}
	return result, nil
}

func (m *mockTracker) FetchIssuesByStates(ctx context.Context, states []string) ([]domain.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stateSet := make(map[string]bool)
	for _, s := range states {
		stateSet[domain.NormalizeState(s)] = true
	}

	var result []domain.Issue
	for _, iss := range m.issues {
		st := iss.State
		if s, ok := m.states[iss.ID]; ok {
			st = s
		}
		if stateSet[domain.NormalizeState(st)] {
			result = append(result, iss)
		}
	}
	return result, nil
}

func (m *mockTracker) FetchIssue(ctx context.Context, id string) (*domain.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, iss := range m.issues {
		if iss.ID == id {
			cp := iss
			cp.Assignee = m.assigned[id]
			if st, ok := m.states[id]; ok {
				cp.State = st
			}
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("issue %q not found", id)
}

func (m *mockTracker) AssignIssue(ctx context.Context, issueID string, identity string) error {
	m.assignCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assigned[issueID] = identity
	return nil
}

func (m *mockTracker) UnassignIssue(ctx context.Context, issueID string) error {
	m.unassignCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.assigned, issueID)
	return nil
}

func (m *mockTracker) setIssueState(id, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[id] = state
}

// --- Mock Launcher ---

type mockLauncher struct {
	mu       sync.Mutex
	exitCode int
	delay    time.Duration
	launches atomic.Int32
}

func newMockLauncher(exitCode int, delay time.Duration) *mockLauncher {
	return &mockLauncher{exitCode: exitCode, delay: delay}
}

func (m *mockLauncher) Launch(ctx context.Context, opts agent.LaunchOpts) (*agent.RunHandle, error) {
	m.launches.Add(1)

	m.mu.Lock()
	ec := m.exitCode
	d := m.delay
	m.mu.Unlock()

	start := time.Now()

	// Create pipe-like readers that produce some output.
	stdoutR, stdoutW := io.Pipe()
	stderrR := strings.NewReader("")

	done := make(chan struct{})
	var once sync.Once

	// Simulate agent work in a goroutine.
	go func() {
		defer once.Do(func() { close(done) })
		defer stdoutW.Close()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		deadline := time.After(d)
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline:
				return
			case <-ticker.C:
				stdoutW.Write([]byte("working...\n"))
			}
		}
	}()

	handle := &agent.RunHandle{
		Stdout: stdoutR,
		Stderr: stderrR,
		Wait: func() (agent.ExitResult, error) {
			select {
			case <-done:
			case <-ctx.Done():
			}
			duration := time.Since(start)
			if ctx.Err() != nil {
				return agent.ExitResult{ExitCode: 1, Duration: duration}, ctx.Err()
			}
			return agent.ExitResult{ExitCode: ec, Duration: duration}, nil
		},
		Cancel: func() {
			once.Do(func() { close(done) })
		},
	}

	return handle, nil
}

func (m *mockLauncher) setExitCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exitCode = code
}

// integrationCfg returns a config that passes ValidateForStartup.
func integrationCfg() *config.ServiceConfig {
	return &config.ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "test-api-key",
		TrackerProjectSlug: "TEST",
		TrackerIdentity:    "test@example.com",
		PollIntervalMs:     100,
		MaxConcurrent:      2,
		MaxRetries:         0,
		TerminalStates:     []string{"Done"},
		AgentCommand:       "echo done",
		StallTimeoutMs:     0,
	}
}

// --- Integration Tests ---

func TestIntegration_ClaimAndDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "PROJ-1", Title: "Task 1", State: "Todo", CreatedAt: &now},
		{ID: "id-2", Identifier: "PROJ-2", Title: "Task 2", State: "Todo", CreatedAt: &now},
	}

	tracker := newMockTracker(issues)
	launcher := newMockLauncher(0, 50*time.Millisecond)

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	cfg := integrationCfg()
	cfg.MaxConcurrent = 2

	o := New(cfg, tracker, ws, testLogger())
	o.launcher = launcher

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Both issues should have been claimed (assigned).
	assert.GreaterOrEqual(t, tracker.assignCalls.Load(), int32(2))

	// Agent should have been launched at least twice.
	assert.GreaterOrEqual(t, launcher.launches.Load(), int32(2))
}

func TestIntegration_ContinuationRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "PROJ-1", Title: "Task 1", State: "Todo", CreatedAt: &now},
	}

	tracker := newMockTracker(issues)
	launcher := newMockLauncher(0, 30*time.Millisecond) // Quick exit with 0.

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	cfg := integrationCfg()
	cfg.PollIntervalMs = 200
	cfg.MaxConcurrent = 1
	cfg.MaxRetries = 2

	o := New(cfg, tracker, ws, testLogger())
	o.launcher = launcher

	// Run for enough time to see continuation retries.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Should have launched more than once due to continuation retry.
	assert.Greater(t, launcher.launches.Load(), int32(1),
		"expected continuation retries to re-launch agent")
}

func TestIntegration_FailureRetryMaxExhausted(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "PROJ-1", Title: "Task 1", State: "Todo", CreatedAt: &now},
	}

	tracker := newMockTracker(issues)
	launcher := newMockLauncher(1, 30*time.Millisecond) // Exit code 1 = failure.

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	cfg := integrationCfg()
	cfg.PollIntervalMs = 200
	cfg.MaxConcurrent = 1
	cfg.MaxRetries = 1
	cfg.MaxRetryBackoffMs = 100 // Very short backoff for test.
	cfg.AgentCommand = "false"

	o := New(cfg, tracker, ws, testLogger())
	o.launcher = launcher

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// After max retries exhausted, claim should be released (unassigned).
	assert.GreaterOrEqual(t, tracker.unassignCalls.Load(), int32(1),
		"expected claim release after max retries exhausted")
}

func TestIntegration_TerminalStateCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "PROJ-1", Title: "Task 1", State: "Todo", CreatedAt: &now},
	}

	tracker := newMockTracker(issues)
	// Agent runs long enough for reconciliation to catch the terminal state change.
	launcher := newMockLauncher(0, 5*time.Second)

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	cfg := integrationCfg()
	cfg.MaxConcurrent = 1
	cfg.AgentCommand = "sleep 10"

	o := New(cfg, tracker, ws, testLogger())
	o.launcher = launcher

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// After a short delay, change the issue to a terminal state.
	go func() {
		time.Sleep(300 * time.Millisecond)
		tracker.setIssueState("id-1", "Done")
	}()

	err = o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// States fetch should have been called (reconciliation).
	assert.GreaterOrEqual(t, tracker.statesFetchCalls.Load(), int32(1),
		"expected reconciliation to fetch states")
}

func TestIntegration_ConcurrencyLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	now := time.Now()
	issues := []domain.Issue{
		{ID: "id-1", Identifier: "PROJ-1", Title: "Task 1", State: "Todo", CreatedAt: &now},
		{ID: "id-2", Identifier: "PROJ-2", Title: "Task 2", State: "Todo", CreatedAt: &now},
		{ID: "id-3", Identifier: "PROJ-3", Title: "Task 3", State: "Todo", CreatedAt: &now},
	}

	tracker := newMockTracker(issues)
	launcher := newMockLauncher(0, 2*time.Second) // Long running.

	tmpDir := t.TempDir()
	ws, err := workspace.NewManager(tmpDir)
	require.NoError(t, err)

	cfg := integrationCfg()
	cfg.MaxConcurrent = 2 // Limit to 2.
	cfg.AgentCommand = "sleep 10"

	o := New(cfg, tracker, ws, testLogger())
	o.launcher = launcher

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Should have launched at most 2 (concurrency limit), not 3.
	assert.LessOrEqual(t, launcher.launches.Load(), int32(2),
		"should respect max_concurrent limit")
}
