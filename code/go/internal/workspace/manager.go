// Package workspace manages per-issue workspace directories.
// See Spec Section 12.
package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneneural/tempad/internal/domain"
)

// HookConfig holds the hook scripts and timeout for workspace lifecycle.
type HookConfig struct {
	AfterCreate string // Script to run after workspace creation (only on new).
	BeforeRun   string // Script to run before each agent run.
	TimeoutMs   int    // Hook timeout in milliseconds.
}

// Manager handles workspace path resolution, creation, and cleanup.
type Manager struct {
	root string // absolute path to workspace root directory
}

// NewManager creates a workspace manager rooted at the given directory.
// The root directory is created automatically if it does not exist.
func NewManager(root string) (*Manager, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create workspace root: %w", err)
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

// Prepare resolves, creates, and runs lifecycle hooks for a workspace.
// Steps: resolve path → ensure dir → after_create (if new) → before_run.
// If after_create fails, the directory is removed. If before_run fails,
// the directory is preserved but an error is returned.
func (m *Manager) Prepare(ctx context.Context, issue domain.Issue, hooks HookConfig) (*domain.Workspace, error) {
	wsPath, err := m.ResolvePath(issue.Identifier)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace path: %w", err)
	}

	isNew, err := m.EnsureDir(wsPath)
	if err != nil {
		return nil, fmt.Errorf("ensure workspace dir: %w", err)
	}

	hookEnv := map[string]string{
		"TEMPAD_ISSUE_ID":      issue.Identifier,
		"TEMPAD_WORKSPACE_DIR": wsPath,
	}

	timeoutMs := hooks.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 30000 // 30s default
	}

	// Run after_create hook only for newly created workspaces.
	if isNew && hooks.AfterCreate != "" {
		_, err := RunHook(ctx, "after_create", hooks.AfterCreate, wsPath, timeoutMs, hookEnv)
		if err != nil {
			// Clean up the newly created directory on after_create failure.
			os.RemoveAll(wsPath)
			return nil, fmt.Errorf("after_create hook: %w", err)
		}
	}

	// Run before_run hook on every prepare.
	if hooks.BeforeRun != "" {
		_, err := RunHook(ctx, "before_run", hooks.BeforeRun, wsPath, timeoutMs, hookEnv)
		if err != nil {
			return nil, fmt.Errorf("before_run hook: %w", err)
		}
	}

	return &domain.Workspace{
		Path:         wsPath,
		WorkspaceKey: domain.SanitizeIdentifier(issue.Identifier),
		CreatedNow:   isNew,
	}, nil
}

// CleanForIssue removes the workspace directory for a specific issue identifier.
// Returns nil if the workspace doesn't exist.
func (m *Manager) CleanForIssue(identifier string) error {
	wsPath, err := m.ResolvePath(identifier)
	if err != nil {
		return fmt.Errorf("resolve path for cleanup: %w", err)
	}

	// Re-verify containment before removal.
	if err := m.verifyContainment(wsPath); err != nil {
		return err
	}

	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return nil // Nothing to clean.
	}

	if err := os.RemoveAll(wsPath); err != nil {
		return fmt.Errorf("remove workspace %q: %w", identifier, err)
	}

	return nil
}

// CleanTerminal removes workspace directories for issues in terminal states.
// Returns a count of cleaned workspaces and any errors encountered.
func (m *Manager) CleanTerminal(issues []domain.Issue) (int, error) {
	cleaned := 0
	var errs []string

	for _, issue := range issues {
		if err := m.CleanForIssue(issue.Identifier); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", issue.Identifier, err))
			continue
		}
		cleaned++
	}

	if len(errs) > 0 {
		return cleaned, fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}

	return cleaned, nil
}

// verifyContainment checks that a path is inside the workspace root.
func (m *Manager) verifyContainment(path string) error {
	rel, err := filepath.Rel(m.root, path)
	if err != nil {
		return fmt.Errorf("path containment check failed: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove %q: outside workspace root", path)
	}
	return nil
}
