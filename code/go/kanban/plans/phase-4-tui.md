# Phase 4: TUI Mode

**Status:** ✅ COMPLETE
**Tickets:** T-P400 to T-P408 (9 tickets)
**Prerequisites:** Phase 2 (tracker client), Phase 3 (workspace manager)
**Goal:** `tempad` (default, no flags) shows a live task board, lets developer select → claim → workspace → IDE.

## Success Criteria

- [ ] `tempad` (no flags) shows live task board from Linear
- [ ] Tasks sorted: priority asc → created_at oldest → identifier
- [ ] Blocked issues marked with `[BLOCKED]`
- [ ] "Available Tasks" and "My Active Tasks" sections
- [ ] Selecting a task claims it, prepares workspace, opens IDE
- [ ] Failed claim shows error, returns to board
- [ ] Manual refresh works (r key)
- [ ] All keybindings work (j/k/Enter/r/d/o/u/q)
- [ ] Task detail view shows all fields
- [ ] Release task (u key) unassigns

## Task List

| # | Ticket | Task | Status | File | Deps |
| --- | -------- | ------ | -------- | ------ | ------ |
| 1 | T-P400 | Claim mechanism (shared) | ✅ Done | `p4-00-claim-mechanism.md` | T-P200 |
| 2 | T-P401 | Bubble Tea app model + messages | ✅ Done | `p4-01-app-model.md` | T-P105, T-P200, T-P300 |
| 3 | T-P402 | Task board view — rendering | ✅ Done | `p4-02-board-view.md` | T-P401 |
| 4 | T-P403 | Keyboard navigation and actions | ✅ Done | `p4-03-keyboard-nav.md` | T-P402 |
| 5 | T-P404 | Task detail view | ✅ Done | `p4-04-detail-view.md` | T-P401 |
| 6 | T-P405 | Poll loop and live refresh | ✅ Done | `p4-05-poll-loop.md` | T-P403 |
| 7 | T-P406 | Task selection flow (claim→workspace→IDE) | ✅ Done | `p4-06-selection-flow.md` | T-P400, T-P302, T-P405 |
| 8 | T-P407 | Release claimed task | ✅ Done | `p4-07-release-task.md` | T-P400, T-P405 |
| 9 | T-P408 | TUI mode entry point | ✅ Done | `p4-08-tui-entrypoint.md` | T-P406, T-P407, T-P303 |

## Dependency Order

```text
T-P400 → T-P401 → {T-P402, T-P404} → T-P403 → T-P405 → T-P406 → T-P407 → T-P408
```

T-P402 and T-P404 can be done in parallel (both need T-P401).

## Parallelization with Phase 5

Phase 4 (TUI) and Phase 5 (Daemon) are independent — they can be developed in parallel once Phases 2 and 3 are complete.

## Research Findings (2026-03-08)

**Key corrections:**

- Add `pollInFlight bool` flag to prevent overlapping polls when Linear API is slow.
- Use model composition with sub-models per view + `viewState` enum (not flat struct).

**Recommendations:**

- Use `lipgloss` for styling, `bubbles/list` for task board, `bubbles/viewport` for detail scrolling.
- Use `glamour` for Markdown rendering of issue descriptions.
- Selection preservation: store issue ID, re-find by ID after poll refresh.
- Use `teatest` for headless TUI testing.

See `handoffs/research-findings.md` for full details.
