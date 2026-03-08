# T-P407: Release claimed task from TUI

**Ticket:** T-P407 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 5.3, 9.5 | **Deps:** T-P400, T-P405

## Description

Allow releasing a claimed task via u key in "My Active Tasks" section.

## Files

- `internal/tui/app.go` (expand)

## Work Items

- [ ] u key on "My Active Tasks" item → confirm → claim.Release() → refresh
- [ ] Only works on issues assigned to current user
- [ ] Show confirmation prompt

## Acceptance Criteria

- [ ] Release unassigns issue
- [ ] Issue moves from "My Active" to "Available" on next refresh
- [ ] Cannot release someone else's task

## Research Notes (2026-03-08)

- Consider a simple inline confirmation ("Release ABC-123? y/n") instead of a separate modal — keeps the UI flow fast.
- After release, trigger an immediate poll refresh to update the board.
