# T-P402: Task board view — rendering

**Ticket:** T-P402 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.2 | **Deps:** T-P401

## Description
Render task board with "Available Tasks" and "My Active Tasks" sections, sorting, blocked markers, Lip Gloss styles.

## Files
- `internal/tui/board.go`
- `internal/tui/styles.go`

## Work Items
- [ ] View() renders two sections: "Available Tasks" (unassigned) and "My Active Tasks" (assigned to user)
- [ ] Each row: identifier, title, priority indicator (P1-P4), state, labels
- [ ] Sorting: priority asc (null last) → created_at oldest → identifier
- [ ] Blocked issues (Todo + non-terminal blockers) → `[BLOCKED]` marker
- [ ] Lip Gloss styles: selected row, priority colors, blocked dimming, headers, footer with keybindings

## Acceptance Criteria
- [ ] Correct sort order
- [ ] Blocked tasks visually distinct
- [ ] Empty state message when no tasks
- [ ] Renders cleanly at 80-column width

## Research Notes (2026-03-08)

- **lipgloss**: Use `lipgloss.NewStyle()` for all styling — it handles colors, borders, padding, margins.
- Consider using the `bubbles/list` component for the task board — it provides built-in filtering, pagination, and keyboard handling out of the box.
- Priority colors: P1 (Urgent) = red, P2 (High) = orange, P3 (Medium) = yellow, P4 (Low) = blue/default.
- Use `lipgloss.Width()` for measuring rendered string width (handles ANSI codes correctly).
