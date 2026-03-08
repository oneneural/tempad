# T-P504: Dispatch loop — claim and spawn

**Ticket:** T-P504 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.3 | **Deps:** T-P503, T-P400

## Description

For each candidate while slots: claim → add to claimed → spawn worker → add to running. Claim fail → skip. Spawn fail → unassign, retry

## Files

- `internal/orchestrator/dispatch.go`

## Work Items

- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria

- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- Dispatch should be non-blocking — if claim fails, skip immediately and try the next candidate.
- The "spawn fail → unassign → retry" path needs careful error handling to avoid orphaned claims if the unassign also fails. Log and continue.
