package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneneural/tempad/internal/domain"
)

func TestCleanForIssue_RemovesDir(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Create a workspace directory.
	wsPath := filepath.Join(root, "ABC-123")
	os.MkdirAll(wsPath, 0755)
	os.WriteFile(filepath.Join(wsPath, "file.txt"), []byte("data"), 0644)

	err := m.CleanForIssue("ABC-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(wsPath); !os.IsNotExist(statErr) {
		t.Error("workspace directory should be removed")
	}
}

func TestCleanForIssue_NonExistent(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Should not error for non-existent workspace.
	err := m.CleanForIssue("DOES-NOT-EXIST")
	if err != nil {
		t.Fatalf("unexpected error for non-existent workspace: %v", err)
	}
}

func TestCleanForIssue_SanitizesIdentifier(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Create a workspace with sanitized name.
	wsPath := filepath.Join(root, "ABC_123")
	os.MkdirAll(wsPath, 0755)

	err := m.CleanForIssue("ABC/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(wsPath); !os.IsNotExist(statErr) {
		t.Error("sanitized workspace directory should be removed")
	}
}

func TestCleanTerminal_MultipleIssues(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Create workspace directories.
	for _, id := range []string{"T-1", "T-2", "T-3"} {
		os.MkdirAll(filepath.Join(root, id), 0755)
	}

	issues := []domain.Issue{
		{Identifier: "T-1"},
		{Identifier: "T-2"},
		{Identifier: "T-3"},
	}

	cleaned, err := m.CleanTerminal(issues)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleaned != 3 {
		t.Errorf("expected 3 cleaned, got %d", cleaned)
	}

	for _, id := range []string{"T-1", "T-2", "T-3"} {
		if _, statErr := os.Stat(filepath.Join(root, id)); !os.IsNotExist(statErr) {
			t.Errorf("workspace %s should be removed", id)
		}
	}
}

func TestCleanTerminal_PartialExistence(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Only create some workspaces.
	os.MkdirAll(filepath.Join(root, "EXISTS-1"), 0755)

	issues := []domain.Issue{
		{Identifier: "EXISTS-1"},
		{Identifier: "MISSING-1"},
	}

	cleaned, err := m.CleanTerminal(issues)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both should count as cleaned (one removed, one already gone).
	if cleaned != 2 {
		t.Errorf("expected 2 cleaned, got %d", cleaned)
	}
}

func TestCleanTerminal_EmptyList(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	cleaned, err := m.CleanTerminal(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("expected 0 cleaned, got %d", cleaned)
	}
}
