# T-P502: Candidate selection and sorting

**Ticket:** T-P502 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.4 | **Deps:** T-P101

## Description
selectCandidates(): filter by fields present, active states, not terminal, unassigned/self, not running/claimed/retry, blocker rule. Sort: priority asc (null last) → created_at → identifier

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

- Candidate selection is a pure function (no side effects) — easy to unit test with table-driven tests.
- Use `sort.SliceStable` for deterministic sorting when priorities are equal.
- The blocker rule check needs the `FetchIssueStatesByIDs` batch query to avoid N+1 API calls.
