package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHook_Success(t *testing.T) {
	dir := t.TempDir()

	result, err := RunHook(context.Background(), "test", "echo hello", dir, 5000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", result.Stdout)
	}
}

func TestRunHook_NonZeroExit(t *testing.T) {
	dir := t.TempDir()

	_, err := RunHook(context.Background(), "failing", "exit 1", dir, 5000, nil)
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
	if !strings.Contains(err.Error(), "failing") {
		t.Errorf("error should mention hook name: %v", err)
	}
}

func TestRunHook_Timeout(t *testing.T) {
	dir := t.TempDir()

	_, err := RunHook(context.Background(), "slow", "sleep 60", dir, 200, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout message, got: %v", err)
	}
}

func TestRunHook_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := RunHook(context.Background(), "cwd", "pwd", dir, 5000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The output should contain the temp directory path.
	if !strings.Contains(result.Stdout, filepath.Base(dir)) {
		t.Errorf("expected cwd to be %s, got %q", dir, result.Stdout)
	}
}

func TestRunHook_Stderr(t *testing.T) {
	dir := t.TempDir()

	result, err := RunHook(context.Background(), "stderr", "echo error >&2", dir, 5000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stderr, "error") {
		t.Errorf("expected stderr to contain 'error', got %q", result.Stderr)
	}
}

func TestRunHook_EmptyScript(t *testing.T) {
	dir := t.TempDir()

	result, err := RunHook(context.Background(), "empty", "", dir, 5000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Stdout != "" || result.Stderr != "" {
		t.Error("expected empty output for empty script")
	}
}

func TestRunHook_EnvironmentVariables(t *testing.T) {
	dir := t.TempDir()

	env := map[string]string{
		"TEMPAD_ISSUE_ID":      "ABC-123",
		"TEMPAD_WORKSPACE_DIR": dir,
	}
	result, err := RunHook(context.Background(), "env", "echo $TEMPAD_ISSUE_ID", dir, 5000, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Stdout, "ABC-123") {
		t.Errorf("expected TEMPAD_ISSUE_ID in output, got %q", result.Stdout)
	}
}

func TestRunHook_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	_, err := RunHook(context.Background(), "create", "touch marker.txt", dir, 5000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "marker.txt")); os.IsNotExist(err) {
		t.Error("expected marker.txt to be created in workspace dir")
	}
}

func TestRunHook_ContextCancellation(t *testing.T) {
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunHook(ctx, "cancelled", "sleep 60", dir, 30000, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
