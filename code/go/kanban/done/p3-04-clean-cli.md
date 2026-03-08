# T-P304: `tempad clean` CLI commands

**Ticket:** T-P304
**Phase:** 3 — Workspace Manager + Hooks
**Status:** 🔲 TODO
**Spec:** Section 18.1
**Deps:** T-P303, T-P204

## Problem

Need CLI commands for workspace cleanup — both tracker-based (all terminal) and manual (specific issue).

## Solution

Expand `tempad clean` placeholder with real implementation.

## Files

- `cmd/tempad/clean.go`

## Work Items

- [ ] `tempad clean` — query tracker for terminal-state issues, remove matching workspaces
- [ ] `tempad clean <identifier>` — remove workspace for specific issue (no tracker needed)
- [ ] Confirmation message for each removal
- [ ] Helpful error when tracker not available for `tempad clean` (no args)

## Acceptance Criteria

- [ ] `tempad clean ABC-123` removes `<root>/ABC-123`
- [ ] `tempad clean` with tracker access removes terminal workspaces
- [ ] `tempad clean` without tracker → helpful error
- [ ] Confirmation message per removal

## Research Notes (2026-03-08)

- Consider adding a `--dry-run` flag that lists workspaces that would be removed without actually deleting them.
- The tracker query for terminal states needs the `FetchIssuesByStates` operation from T-P204 — ensure state name matching is case-insensitive.
- Use `project.slug` (not `slugId`) for the Linear query — validated in Phase 2 research.
