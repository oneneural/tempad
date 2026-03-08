# T-P508: Agent output + stall detection

**Ticket:** T-P508 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 11.4, 10.7A | **Deps:** T-P507

## Description
OutputMonitor: tee to log file, atomic lastOutputAt on each read, optional JSON lines parsing, tolerate no output. LastOutputAt() for stall detection

## Files
- `internal/agent/output.go`

## Work Items
- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria
- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- Use `atomic.Value` or `atomic.Int64` (Unix timestamp) for `lastOutputAt` — avoids mutex contention on hot read path.
- The stall detection threshold should come from config (`agent.stall_timeout_ms`), not hard-coded.
- Consider using `bufio.Scanner` for line-by-line output processing — simpler than raw Read() calls and handles line boundaries correctly.
