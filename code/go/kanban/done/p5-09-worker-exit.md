# T-P509: Worker exit handling

**Ticket:** T-P509 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 19.6 | **Deps:** T-P501, T-P510

## Description

handleWorkerExit: remove from running. Exit 0 → completed + continuation retry (1s). Exit != 0 → failure retry (exponential backoff)

## Files

- `internal/orchestrator/orchestrator.go`

## Work Items

- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria

- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- Continuation retry (exit 0, 1s delay) does NOT count toward `max_retries` — it's a successful completion with more work to do.
- Failure retry uses exponential backoff: `min(10000 * 2^(n-1), max_backoff_ms)`.
- Consider using `cenkalti/backoff/v4` library for robust exponential backoff with jitter.
