# T-P408: TUI mode entry point

**Ticket:** T-P408 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.1 | **Deps:** T-P406, T-P407, T-P303

## Description

Wire up TUI mode as the default when no --daemon flag is provided.

## Files

- `cmd/tempad/main.go` (expand root command)

## Work Items

- [ ] No --daemon → TUI mode
- [ ] Load + merge + validate config
- [ ] Create tracker client, workspace manager
- [ ] Startup terminal workspace cleanup
- [ ] Create tea.Program with Model, p.Run()
- [ ] Graceful exit on quit

## Acceptance Criteria

- [ ] `tempad` launches TUI with task board from Linear
- [ ] Ctrl+C exits cleanly
- [ ] Startup validation failure → exit 1

## Research Notes (2026-03-08)

- Use `tea.WithAltScreen()` option for full-screen TUI that restores terminal on exit.
- Use `tea.WithMouseCellMotion()` if mouse support is desired (for viewport scrolling).
- Consider `teatest` (charmbracelet/x/exp/teatest) for headless testing of the full TUI startup/interaction flow.
