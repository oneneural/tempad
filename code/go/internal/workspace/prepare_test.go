package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneneural/tempad/internal/domain"
)

func testIssue(id string) domain.Issue {
	return domain.Issue{
		ID:         "uuid-" + id,
		Identifier: id,
		Title:      "Test Issue " + id,
	}
}

func TestPrepare_NewWorkspace(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	ws, err := m.Prepare(context.Background(), testIssue("ABC-123"), HookConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ws.CreatedNow {
		t.Error("expected CreatedNow=true for new workspace")
	}
	if ws.WorkspaceKey != "ABC-123" {
		t.Errorf("expected WorkspaceKey=ABC-123, got %s", ws.WorkspaceKey)
	}
	if ws.Path != filepath.Join(root, "ABC-123") {
		t.Errorf("unexpected path: %s", ws.Path)
	}

	// Directory should exist.
	info, err := os.Stat(ws.Path)
	if err != nil || !info.IsDir() {
		t.Error("workspace directory should exist")
	}
}

func TestPrepare_ExistingWorkspace(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Pre-create the directory.
	os.MkdirAll(filepath.Join(root, "ABC-123"), 0755)

	ws, err := m.Prepare(context.Background(), testIssue("ABC-123"), HookConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.CreatedNow {
		t.Error("expected CreatedNow=false for existing workspace")
	}
}

func TestPrepare_AfterCreateHook(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	hooks := HookConfig{
		AfterCreate: "touch after_create.marker",
		TimeoutMs:   5000,
	}

	ws, err := m.Prepare(context.Background(), testIssue("HOOK-1"), hooks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marker := filepath.Join(ws.Path, "after_create.marker")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("after_create hook should have created marker file")
	}
}

func TestPrepare_AfterCreateSkippedOnExisting(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Pre-create workspace.
	os.MkdirAll(filepath.Join(root, "HOOK-2"), 0755)

	hooks := HookConfig{
		AfterCreate: "touch after_create.marker",
		TimeoutMs:   5000,
	}

	ws, err := m.Prepare(context.Background(), testIssue("HOOK-2"), hooks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marker := filepath.Join(ws.Path, "after_create.marker")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("after_create hook should NOT run on existing workspace")
	}
}

func TestPrepare_AfterCreateFailure_RemovesDir(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	hooks := HookConfig{
		AfterCreate: "exit 1",
		TimeoutMs:   5000,
	}

	_, err := m.Prepare(context.Background(), testIssue("FAIL-1"), hooks)
	if err == nil {
		t.Fatal("expected error from failing after_create hook")
	}

	// Directory should be cleaned up.
	wsPath := filepath.Join(root, "FAIL-1")
	if _, statErr := os.Stat(wsPath); !os.IsNotExist(statErr) {
		t.Error("workspace directory should be removed after after_create failure")
	}
}

func TestPrepare_BeforeRunHook(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	hooks := HookConfig{
		BeforeRun: "touch before_run.marker",
		TimeoutMs: 5000,
	}

	ws, err := m.Prepare(context.Background(), testIssue("HOOK-3"), hooks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marker := filepath.Join(ws.Path, "before_run.marker")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("before_run hook should have created marker file")
	}
}

func TestPrepare_BeforeRunFailure_PreservesDir(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	hooks := HookConfig{
		BeforeRun: "exit 1",
		TimeoutMs: 5000,
	}

	_, err := m.Prepare(context.Background(), testIssue("FAIL-2"), hooks)
	if err == nil {
		t.Fatal("expected error from failing before_run hook")
	}

	// Directory should be preserved even though before_run failed.
	wsPath := filepath.Join(root, "FAIL-2")
	if _, statErr := os.Stat(wsPath); os.IsNotExist(statErr) {
		t.Error("workspace directory should be preserved after before_run failure")
	}
}

func TestPrepare_BothHooks(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	hooks := HookConfig{
		AfterCreate: "touch step1.marker",
		BeforeRun:   "touch step2.marker",
		TimeoutMs:   5000,
	}

	ws, err := m.Prepare(context.Background(), testIssue("BOTH-1"), hooks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ws.Path, "step1.marker")); os.IsNotExist(err) {
		t.Error("after_create marker missing")
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "step2.marker")); os.IsNotExist(err) {
		t.Error("before_run marker missing")
	}
}
