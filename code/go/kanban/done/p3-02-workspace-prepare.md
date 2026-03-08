# T-P302: Workspace Prepare lifecycle

**Ticket:** T-P302
**Phase:** 3 — Workspace Manager + Hooks
**Status:** 🔲 TODO
**Spec:** Section 12.2, 12.3, 12.4
**Deps:** T-P300, T-P301

## Problem

Need a single `Prepare()` method that handles the full workspace lifecycle: resolve → create → after_create hook → before_run hook.

## Solution

`Prepare(ctx, issue, hookConfig)` orchestrating path resolution, directory creation, and hook execution.

## Files

- `internal/workspace/manager.go` (expand)

## Work Items

- [ ] `Prepare(ctx, issue, hookConfig) (*domain.Workspace, error)`
- [ ] Steps: resolve path → ensure dir → if new + after_create hook → run (failure removes dir) → before_run hook → run (failure returns error)
- [ ] Return `domain.Workspace{Path, WorkspaceKey, CreatedNow}`
- [ ] Validate cwd == workspace_path before returning

## Acceptance Criteria

- [ ] New workspace → after_create → before_run → success
- [ ] Existing workspace → after_create skipped → before_run → success
- [ ] after_create failure → directory removed, error returned
- [ ] before_run failure → error, directory preserved
- [ ] Integration test with real filesystem

## Research Notes (2026-03-08)

- Use `t.TempDir()` for integration tests — Go auto-cleans it.
- The `after_create` hook cleanup (remove dir on failure) should use `os.RemoveAll` with the same `filepath.Rel()` safety check from T-P300.
- Consider adding a `CreatedNow` flag to `domain.Workspace` so callers know if `after_create` ran — useful for idempotency logging.
