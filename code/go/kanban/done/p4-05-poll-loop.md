# T-P405: Poll loop and live refresh

**Ticket:** T-P405 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.4 | **Deps:** T-P403

## Description

Implement polling loop that refreshes the task list at configured intervals.

## Files

- `internal/tui/app.go` (expand Update)

## Work Items

- [ ] `pollCmd()` calls `tracker.FetchCandidateIssues()` → PollResultMsg
- [ ] `tickCmd` via `tea.Tick(pollInterval, …)`
- [ ] On r key → immediate poll
- [ ] "Refreshing..." indicator, error shown inline
- [ ] No duplicate concurrent polls

## Acceptance Criteria

- [ ] Task list updates every polling.interval_ms
- [ ] Manual refresh works
- [ ] Selection preserved after refresh
- [ ] Poll error shown as status, board still usable

## Research Notes (2026-03-08)

- **CRITICAL — Polling deduplication**: Add a `pollInFlight bool` flag to prevent overlapping polls when the Linear API is slow. In `Update`: only fire poll Cmd if `!m.pollInFlight`. Set flag on poll start, clear on `PollResultMsg` receipt.
- Use `tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })` — NOT `time.Ticker` which would require goroutine management.
- The tick returns a new tick command, creating a self-renewing loop. This is the idiomatic Bubble Tea approach.
