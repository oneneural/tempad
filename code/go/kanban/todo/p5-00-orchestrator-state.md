# T-P500: Orchestrator runtime state

**Ticket:** T-P500 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 4.1.8, 10.2.1 | **Deps:** T-P101, T-P200, T-P300, T-P400

## Description
Orchestrator struct with state, channels (workerResults, retryTimers, configReload), WorkerResult and RetrySignal structs

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

- **Channel buffer sizes**: Use buffered channels for worker results: `make(chan WorkerResult, maxConcurrent)`. Unbuffered channels can cause goroutine leaks when the orchestrator is shutting down and not reading from the channel.
- Use `sync.Map` or a mutex-protected map for the `running` and `claimed` state maps — multiple goroutines will read/write them concurrently.
- Consider using `errgroup` (golang.org/x/sync/errgroup) for managing worker goroutine lifecycle.
