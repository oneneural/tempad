# T-P403: Task board — keyboard navigation and actions

**Ticket:** T-P403 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.5 | **Deps:** T-P402

## Description
Implement all keybindings for task board navigation and actions.

## Files
- `internal/tui/keys.go`
- `internal/tui/app.go` (Update method)

## Work Items
- [ ] j/↓ down, k/↑ up, Enter select, r refresh, d details, o open URL, u release, q/Ctrl+C quit
- [ ] Selection state preserved across refresh (match by issue ID)
- [ ] Cursor wraps or stops at boundaries

## Acceptance Criteria
- [ ] All keybindings work
- [ ] Refresh doesn't reset cursor
- [ ] q exits cleanly
- [ ] o opens URL (or shows error)

## Research Notes (2026-03-08)

- **Selection preservation**: Store the selected issue's ID (not index). After poll refresh, iterate the new list to find the matching ID and restore the cursor position. If the issue is gone, select the nearest index.
- Use `exec.Command("open", url)` on macOS, `exec.Command("xdg-open", url)` on Linux for the `o` key URL opening.
- Consider using `bubbles/key` for keybinding definitions — provides help text generation for free.
