# T-P503: Concurrency control

**Ticket:** T-P503 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.5 | **Deps:** T-P502

## Description
availableSlots() and stateSlotAvailable() for global and per-state concurrency limits. Invalid per-state entries ignored

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

- Per-state concurrency limits should be validated at config load time — invalid state names logged and ignored at runtime.
- Consider a `Semaphore` pattern using buffered channels (`make(chan struct{}, limit)`) for clean concurrency control.
