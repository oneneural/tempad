package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	root := t.TempDir()
	m, err := NewManager(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Root() != root {
		t.Errorf("expected root=%s, got %s", root, m.Root())
	}
}

func TestNewManager_NonExistent(t *testing.T) {
	_, err := NewManager("/nonexistent/path/for/testing")
	if err == nil {
		t.Fatal("expected error for non-existent root")
	}
}

func TestNewManager_NotADirectory(t *testing.T) {
	f, err := os.CreateTemp("", "tempad-test-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	_, err = NewManager(f.Name())
	if err == nil {
		t.Fatal("expected error for file (not directory) root")
	}
}

func TestResolvePath_Simple(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path, err := m.ResolvePath("ABC-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, "ABC-123")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestResolvePath_Sanitized(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path, err := m.ResolvePath("ABC/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, "ABC_123")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestResolvePath_TraversalAttack_DotDot(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// "../../etc/passwd" sanitizes to ".._.._etc_passwd" (dots preserved, slashes replaced).
	// This stays under root because filepath.Join resolves it as a child.
	path, err := m.ResolvePath("../../etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, ".._.._etc_passwd")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestResolvePath_TraversalAttack_AbsolutePath(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	// Absolute path gets sanitized (/ → _).
	path, err := m.ResolvePath("/etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, "_etc_passwd")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestResolvePath_TraversalAttack_Backslash(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path, err := m.ResolvePath(`..\..\windows\system32`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, ".._.._windows_system32")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestResolvePath_Unicode(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path, err := m.ResolvePath("слово")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, "_____")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestResolvePath_Empty(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	_, err := m.ResolvePath("")
	if err == nil {
		t.Fatal("expected error for empty identifier")
	}
}

func TestEnsureDir_CreatesNew(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path := filepath.Join(root, "new-workspace")
	isNew, err := m.EnsureDir(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true for new directory")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestEnsureDir_Existing(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path := filepath.Join(root, "existing")
	os.MkdirAll(path, 0755)

	isNew, err := m.EnsureDir(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isNew {
		t.Error("expected isNew=false for existing directory")
	}
}

func TestEnsureDir_FileExists(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	path := filepath.Join(root, "file-exists")
	os.WriteFile(path, []byte("not a dir"), 0644)

	_, err := m.EnsureDir(path)
	if err == nil {
		t.Fatal("expected error when file exists at path")
	}
}

func TestEnsureDir_OutsideRoot(t *testing.T) {
	root := t.TempDir()
	m, _ := NewManager(root)

	_, err := m.EnsureDir("/tmp/outside-root-test")
	if err == nil {
		t.Fatal("expected error for path outside root")
	}
}
