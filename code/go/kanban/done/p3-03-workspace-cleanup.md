# T-P303: Workspace cleanup (terminal + manual)

**Ticket:** T-P303
**Phase:** 3 — Workspace Manager + Hooks
**Status:** 🔲 TODO
**Spec:** Section 10.8, 12.5, 18.1
**Deps:** T-P300, T-P301

## Problem

Need to clean up workspaces for terminal-state issues (startup cleanup) and manual cleanup.

## Solution

`CleanForIssue()` and `CleanTerminal()` methods with before_remove hook support.

## Files

- `internal/workspace/cleanup.go`

## Work Items

- [ ] `CleanForIssue(ctx, identifier)`: resolve → before_remove hook (failure logged/ignored) → `os.RemoveAll` if under root
- [ ] `CleanTerminal(ctx, terminalIssues)`: iterate, call CleanForIssue, log each, continue on failures
- [ ] Never remove paths outside workspace root

## Acceptance Criteria

- [ ] Removes existing workspace directory
- [ ] No-op if doesn't exist
- [ ] before_remove hook runs before removal
- [ ] before_remove failure doesn't prevent removal
- [ ] Never removes outside root
- [ ] Unit tests pass

## Research Notes (2026-03-08)

- **Path safety**: All `os.RemoveAll` calls must re-verify path is under workspace root using `filepath.Rel()` — never trust a cached path.
- The `before_remove` hook runs in the workspace directory (set cwd), so the hook script can do final cleanup (e.g., git stash).
- Consider logging the total size of removed directories for operational awareness.
