package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// Ignore known background goroutines from test infrastructure.
		goleak.IgnoreTopFunction("testing.(*T).Run"),
	)
}

func TestLeak_ShutdownCleansUp(t *testing.T) {
	cfg := testConfig()
	o := New(cfg, nil, nil, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := o.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// After Run returns, all goroutines should be cleaned up.
	// goleak.VerifyTestMain will catch any leaked goroutines.
}

func TestLeak_RetryTimerCancellation(t *testing.T) {
	cfg := testConfig()
	cfg.MaxRetries = 10 // High enough so scheduleRetry doesn't try to release.
	o := New(cfg, nil, nil, testLogger())

	ctx, cancel := context.WithCancel(context.Background())

	// Schedule a retry with a long delay.
	o.scheduleRetry(ctx, "id-1", "PROJ-1", 1, false, "test error")

	// Cancel immediately — timer should not leak.
	cancel()

	// Give the timer a moment to fire (it shouldn't cause issues).
	time.Sleep(50 * time.Millisecond)

	// Cleanup timer.
	if timer, ok := o.activeTimers["id-1"]; ok {
		timer.Stop()
	}
}
