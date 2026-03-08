package agent

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubprocessLauncher_Success(t *testing.T) {
	launcher := NewSubprocessLauncher()
	dir := t.TempDir()

	handle, err := launcher.Launch(context.Background(), LaunchOpts{
		Command:       "echo hello",
		WorkspacePath: dir,
		Prompt:        "test prompt",
		PromptMethod:  "env",
		Env:           map[string]string{"TEMPAD_ISSUE_ID": "test-1"},
	})
	require.NoError(t, err)
	require.NotNil(t, handle)

	// Read stdout.
	out, err := io.ReadAll(handle.Stdout)
	require.NoError(t, err)
	assert.Contains(t, string(out), "hello")

	// Wait for exit.
	result, err := handle.Wait()
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestSubprocessLauncher_NonZeroExit(t *testing.T) {
	launcher := NewSubprocessLauncher()
	dir := t.TempDir()

	handle, err := launcher.Launch(context.Background(), LaunchOpts{
		Command:       "exit 42",
		WorkspacePath: dir,
		Prompt:        "test",
		PromptMethod:  "env",
	})
	require.NoError(t, err)

	result, err := handle.Wait()
	require.NoError(t, err) // Non-zero exit is a result, not an error.
	assert.Equal(t, 42, result.ExitCode)
}

func TestSubprocessLauncher_FileDelivery(t *testing.T) {
	launcher := NewSubprocessLauncher()
	dir := t.TempDir()

	handle, err := launcher.Launch(context.Background(), LaunchOpts{
		Command:       "cat PROMPT.md",
		WorkspacePath: dir,
		Prompt:        "Work on PROJ-42",
		PromptMethod:  "file",
	})
	require.NoError(t, err)

	out, _ := io.ReadAll(handle.Stdout)
	result, err := handle.Wait()
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "Work on PROJ-42", string(out))
}

func TestSubprocessLauncher_Cancel(t *testing.T) {
	launcher := NewSubprocessLauncher()
	dir := t.TempDir()

	handle, err := launcher.Launch(context.Background(), LaunchOpts{
		Command:       "sleep 60",
		WorkspacePath: dir,
		Prompt:        "test",
		PromptMethod:  "env",
	})
	require.NoError(t, err)

	// Give the process a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	handle.Cancel()

	done := make(chan ExitResult, 1)
	go func() {
		result, _ := handle.Wait()
		done <- result
	}()

	select {
	case result := <-done:
		// Process was killed — should finish quickly.
		assert.True(t, result.Duration < 10*time.Second, "process should have been killed quickly")
	case <-time.After(10 * time.Second):
		t.Fatal("cancel did not terminate process within 10s")
	}
}
