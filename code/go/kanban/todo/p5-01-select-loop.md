# T-P501: Orchestrator main select loop

**Ticket:** T-P501 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.3, Arch 5.3 | **Deps:** T-P500

## Description
Run(ctx) with select over ctx.Done/ticker/workerResults/retryTimers/configReload. Graceful shutdown: cancel workers, wait, release claims. SIGINT/SIGTERM handling

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

- **Standard Go pattern**: The for-select over `ctx.Done/ticker.C/workerResults/retryTimers/configReload` is the canonical Go orchestrator pattern.
- **Graceful shutdown sequence**: Stop ticker → cancel all worker contexts → wait with configurable timeout → release all claims → exit.
- **SIGINT/SIGTERM**: Use `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` — cleaner than manual channel management.
- All channel reads in the select must handle the case where the channel is closed (e.g., on shutdown).
