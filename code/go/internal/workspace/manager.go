// Package workspace manages per-issue workspace directories.
// See Spec Section 12.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneneural/tempad/internal/domain"
)

// Manager handles workspace path resolution, creation, and cleanup.
type Manager struct {
	root string // absolute path to workspace root directory
}

// NewManager creates a workspace manager rooted at the given directory.
// The root directory must exist and be a directory.
func NewManager(root string) (*Manager, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace root %q is not a directory", abs)
	}

	return &Manager{root: abs}, nil
}

// Root returns the absolute path to the workspace root.
func (m *Manager) Root() string {
	return m.root
}

// ResolvePath returns the absolute workspace path for a given issue identifier.
// It sanitizes the identifier and verifies the result stays within the root.
func (m *Manager) ResolvePath(identifier string) (string, error) {
	sanitized := domain.SanitizeIdentifier(identifier)
	if sanitized == "" {
		return "", fmt.Errorf("empty identifier after sanitization")
	}

	candidate := filepath.Join(m.root, sanitized)

	// Verify path containment using filepath.Rel.
	rel, err := filepath.Rel(m.root, candidate)
	if err != nil {
		return "", fmt.Errorf("path safety check failed: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace root", identifier)
	}

	return candidate, nil
}

// EnsureDir creates the workspace directory if it doesn't exist.
// Returns true if the directory was newly created, false if it existed.
func (m *Manager) EnsureDir(path string) (isNew bool, err error) {
	// Verify containment before any filesystem operation.
	rel, err := filepath.Rel(m.root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false, fmt.Errorf("path %q is outside workspace root", path)
	}

	info, err := os.Stat(path)
	if err == nil {
		// Path exists — must be a directory.
		if !info.IsDir() {
			return false, fmt.Errorf("path %q exists but is not a directory", path)
		}
		return false, nil
	}

	if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat workspace path: %w", err)
	}

	// Create directory.
	if err := os.MkdirAll(path, 0755); err != nil {
		return false, fmt.Errorf("create workspace directory: %w", err)
	}

	return true, nil
}
