package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWatcherLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// validWorkflowContent returns a minimal valid WORKFLOW.md.
func validWorkflowContent(slug string) string {
	return `---
tracker:
  kind: linear
  api_key: test-key
  project_slug: ` + slug + `
  identity: test@example.com
agent:
  command: echo done
---
Prompt body here.
`
}

func TestWatcher_DetectsChange(t *testing.T) {
	if testing.Short() {
		t.Skip("file watcher test")
	}

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "WORKFLOW.md")

	// Write initial file.
	err := os.WriteFile(workflowPath, []byte(validWorkflowContent("PROJ")), 0644)
	require.NoError(t, err)

	cli := &CLIFlags{WorkflowPath: workflowPath}
	w, err := NewWatcher(workflowPath, cli, testWatcherLogger())
	require.NoError(t, err)
	defer w.Stop()

	w.Start()

	// Wait a bit for watcher to be ready.
	time.Sleep(100 * time.Millisecond)

	// Modify the file.
	err = os.WriteFile(workflowPath, []byte(validWorkflowContent("PROJ2")), 0644)
	require.NoError(t, err)

	// Should receive reload within 2s.
	select {
	case cfg := <-w.ReloadCh():
		assert.Equal(t, "PROJ2", cfg.TrackerProjectSlug)
	case <-time.After(2 * time.Second):
		t.Fatal("expected config reload after file change")
	}
}

func TestWatcher_Debounce(t *testing.T) {
	if testing.Short() {
		t.Skip("file watcher test")
	}

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "WORKFLOW.md")

	err := os.WriteFile(workflowPath, []byte(validWorkflowContent("PROJ")), 0644)
	require.NoError(t, err)

	cli := &CLIFlags{WorkflowPath: workflowPath}
	w, err := NewWatcher(workflowPath, cli, testWatcherLogger())
	require.NoError(t, err)
	defer w.Stop()

	w.Start()
	time.Sleep(100 * time.Millisecond)

	// Rapid edits.
	for i := 0; i < 5; i++ {
		err = os.WriteFile(workflowPath, []byte(validWorkflowContent("FINAL")), 0644)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Should receive exactly one reload with the final value.
	select {
	case cfg := <-w.ReloadCh():
		assert.Equal(t, "FINAL", cfg.TrackerProjectSlug)
	case <-time.After(2 * time.Second):
		t.Fatal("expected config reload after rapid edits")
	}

	// Drain any additional (shouldn't be any).
	select {
	case <-w.ReloadCh():
		// At most one extra is acceptable due to timing.
	case <-time.After(300 * time.Millisecond):
		// No extras — good.
	}
}

func TestWatcher_InvalidConfigKeepsCurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("file watcher test")
	}

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "WORKFLOW.md")

	err := os.WriteFile(workflowPath, []byte(validWorkflowContent("PROJ")), 0644)
	require.NoError(t, err)

	cli := &CLIFlags{WorkflowPath: workflowPath}
	w, err := NewWatcher(workflowPath, cli, testWatcherLogger())
	require.NoError(t, err)
	defer w.Stop()

	w.Start()
	time.Sleep(100 * time.Millisecond)

	// Write invalid content (missing required fields).
	err = os.WriteFile(workflowPath, []byte("---\ninvalid: true\n---\n"), 0644)
	require.NoError(t, err)

	// Should NOT receive a reload (invalid config is dropped).
	select {
	case <-w.ReloadCh():
		t.Fatal("should not reload invalid config")
	case <-time.After(1500 * time.Millisecond):
		// Good — no reload sent.
	}
}
