# T-P510: Retry scheduling and backoff

**Ticket:** T-P510 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.6 | **Deps:** T-P501

## Description
scheduleRetry: cancel existing timer, compute delay (continuation 1s, failure min(10000*2^(n-1), max)). handleRetry: fetch issue, check eligibility, max_retries, dispatch/requeue. Continuation doesn't count toward max

## Files
- `internal/orchestrator/retry.go`

## Work Items
- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria
- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- **CRITICAL**: `time.AfterFunc` callbacks must check `ctx.Err() != nil` before modifying state — they can fire after shutdown.
- Cancel existing timer before scheduling new one (`timer.Stop()` returns false if already fired).
- Consider `cenkalti/backoff/v4` for exponential backoff with jitter.
- Add jitter (±20%) to prevent thundering herd when multiple retries fire simultaneously.
