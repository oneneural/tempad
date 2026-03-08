# T-P802: Goroutine leak detection

**Ticket:** T-P802 | **Phase:** 8 — Testing + Hardening | **Status:** 🔲 TODO
**Deps:** T-P800

## Description

goleak in orchestrator tests. Verify shutdown, cancellation, and retry timer edge cases leave no leaks.

## Files

- `internal/orchestrator/leak_test.go`

## Acceptance Criteria

- [ ] goleak reports zero leaks in all scenarios

## Research Notes (2026-03-08)

- Use `goleak.VerifyNone(t)` either per-test or in `TestMain(m *testing.M)` for the orchestrator package.
- Common leak sources: unbuffered channels not read after shutdown, `time.AfterFunc` callbacks holding references, goroutines blocked on closed channels.
- Ensure worker result channels are buffered (`make(chan WorkerResult, maxConcurrent)`) to prevent leaks during shutdown.
