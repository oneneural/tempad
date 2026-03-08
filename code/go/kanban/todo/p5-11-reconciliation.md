# T-P511: Active run reconciliation

**Ticket:** T-P511 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.7 | **Deps:** T-P508, T-P303

## Description
Part A: stall detection (lastOutputAt > threshold → cancel + retry). Part B: tracker state refresh (terminal → kill+clean, active → update, other → kill). Fetch fail → keep running

## Files
- `internal/orchestrator/reconcile.go`

## Work Items
- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria
- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- **Stall detection**: Compare `time.Since(worker.LastOutputAt())` against threshold on each reconciliation tick.
- **Tracker state refresh**: Batch fetch states using `FetchIssueStatesByIDs` to avoid N+1 API calls.
- "Fetch fail → keep running" is the correct safe default — don't kill workers just because the API is temporarily unreachable.
- Terminal state → kill + clean should use the same process group kill pattern from T-P507.
