# T-P404: Task detail view

**Ticket:** T-P404 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.5 | **Deps:** T-P401

## Description
Full-screen detail view for a selected task showing all issue fields.

## Files
- `internal/tui/detail.go`

## Work Items
- [ ] Display: identifier, title, state, priority, description (wrapped), labels, blockers (with states), URL, timestamps
- [ ] Esc/Backspace returns to board
- [ ] Scrollable if content exceeds terminal height

## Acceptance Criteria
- [ ] All issue fields displayed
- [ ] Long descriptions wrap correctly
- [ ] Escape returns to board

## Research Notes (2026-03-08)

- Use `bubbles/viewport` component for scrollable content — handles mouse wheel and keyboard scrolling.
- Description text should be word-wrapped using `lipgloss.NewStyle().Width(termWidth - padding)` or the `wordwrap` library from Charm.
- Consider Markdown rendering for descriptions using `glamour` (charmbracelet/glamour) — Linear descriptions are often Markdown formatted.
