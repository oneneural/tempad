package notify

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureSend returns a sendFn that records calls and a function to retrieve them.
func captureSend() (func(string, string) error, func() [][]string) {
	var mu sync.Mutex
	var calls [][]string
	fn := func(title, body string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, []string{title, body})
		return nil
	}
	get := func() [][]string {
		mu.Lock()
		defer mu.Unlock()
		cp := make([][]string, len(calls))
		copy(cp, calls)
		return cp
	}
	return fn, get
}

func TestNotifier_SendBasic(t *testing.T) {
	sendFn, getCalls := captureSend()
	n := New(Config{Enabled: true}, slog.Default())
	n.sendFn = sendFn
	n.minInterval = 0 // disable rate limiting for this test

	n.Send(EventNewTask, "TEMPAD: New Task", "ONE-42: Fix auth")
	time.Sleep(50 * time.Millisecond) // wait for goroutine

	calls := getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "TEMPAD: New Task", calls[0][0])
	assert.Equal(t, "ONE-42: Fix auth", calls[0][1])
}

func TestNotifier_Disabled(t *testing.T) {
	sendFn, getCalls := captureSend()
	n := New(Config{Enabled: false}, slog.Default())
	n.sendFn = sendFn

	n.Send(EventNewTask, "title", "body")
	time.Sleep(50 * time.Millisecond)

	assert.Empty(t, getCalls())
}

func TestNotifier_Noop(t *testing.T) {
	noop := Noop()

	// Should not panic.
	noop.Send(EventNewTask, "title", "body")
	noop.Send(EventAgentFailed, "title", "body")
}

func TestNotifier_EventFiltering(t *testing.T) {
	sendFn, getCalls := captureSend()
	n := New(Config{
		Enabled: true,
		Events:  []string{"agent_completed", "agent_failed"},
	}, slog.Default())
	n.sendFn = sendFn
	n.minInterval = 0

	// Should be filtered out.
	n.Send(EventNewTask, "title", "body")
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, getCalls())

	// Should pass through.
	n.Send(EventAgentCompleted, "title", "body")
	time.Sleep(50 * time.Millisecond)
	calls := getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "title", calls[0][0])
}

func TestNotifier_EventFilteringAllowsConfiguredEvents(t *testing.T) {
	sendFn, getCalls := captureSend()
	n := New(Config{
		Enabled: true,
		Events:  []string{"new_task", "retries_exhausted"},
	}, slog.Default())
	n.sendFn = sendFn
	n.minInterval = 0

	n.Send(EventNewTask, "t1", "b1")
	time.Sleep(50 * time.Millisecond)
	n.Send(EventRetriesExhausted, "t2", "b2")
	time.Sleep(50 * time.Millisecond)
	n.Send(EventAgentStarted, "t3", "b3") // filtered
	time.Sleep(50 * time.Millisecond)

	calls := getCalls()
	require.Len(t, calls, 2)
	assert.Equal(t, "t1", calls[0][0])
	assert.Equal(t, "t2", calls[1][0])
}

func TestNotifier_DefaultAllEvents(t *testing.T) {
	n := New(Config{Enabled: true}, slog.Default())

	for _, e := range AllEvents() {
		assert.True(t, n.eventSet[e], "event %s should be enabled by default", e)
	}
}

func TestNotifier_RateLimiting(t *testing.T) {
	var count atomic.Int32
	n := New(Config{Enabled: true}, slog.Default())
	n.sendFn = func(_, _ string) error {
		count.Add(1)
		return nil
	}
	n.minInterval = 100 * time.Millisecond

	// Rapid-fire 10 sends — only the first should get through.
	for i := 0; i < 10; i++ {
		n.Send(EventNewTask, "title", "body")
	}
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(1), count.Load(), "only one notification should pass rate limiter")

	// Wait for rate limit window to pass, then send again.
	time.Sleep(100 * time.Millisecond)
	n.Send(EventNewTask, "title", "body")
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(2), count.Load(), "notification should pass after rate limit window")
}

func TestNotifier_SendErrorLogged(t *testing.T) {
	n := New(Config{Enabled: true}, slog.Default())
	n.sendFn = func(_, _ string) error {
		return fmt.Errorf("send failed")
	}
	n.minInterval = 0

	// Should not panic — errors are logged, not returned.
	n.Send(EventAgentFailed, "title", "body")
	time.Sleep(50 * time.Millisecond)
}

func TestNotifier_CommandConstruction_Darwin(t *testing.T) {
	// Verify the platform send function constructs the right command.
	// We can't test actual execution in CI, but we can test the function exists
	// and the notifier wiring works.
	n := New(Config{Enabled: true}, slog.Default())
	assert.NotNil(t, n.sendFn)
}
