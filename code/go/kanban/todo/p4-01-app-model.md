# T-P401: Bubble Tea app model and message types

**Ticket:** T-P401 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.1 | **Deps:** T-P105, T-P200, T-P300

## Description
Define Bubble Tea Model struct, all message types, and Init() command.

## Files
- `internal/tui/app.go`
- `internal/tui/messages.go`

## Work Items
- [ ] `Model` struct implementing `tea.Model` (config, tracker, workspace, claimer, task list, cursor, view state)
- [ ] Message types: PollResultMsg, ClaimResultMsg, WorkspaceReadyMsg, IDEOpenedMsg, ConfigReloadMsg, tickMsg
- [ ] `Init()` → `tea.Batch(pollCmd, tickCmd)`
- [ ] Tick interval from `config.PollIntervalMs`

## Acceptance Criteria
- [ ] Model compiles and implements `tea.Model`
- [ ] Init returns poll + tick commands
- [ ] All message types defined

## Research Notes (2026-03-08)

- **Model composition**: Use embedded sub-models for each view (board view, detail view) with a `viewState` enum (`viewBoard`, `viewDetail`) to switch — don't flatten everything into one struct. This follows Bubble Tea community best practices for multi-view apps.
- **BubbleUp pattern**: Sub-models return commands that propagate status messages up to the parent model.
- Handle `tea.WindowSizeMsg` to store terminal dimensions and pass to sub-views for responsive layout.
- Consider adding an `errMsg` type for consistent error propagation from async commands.
