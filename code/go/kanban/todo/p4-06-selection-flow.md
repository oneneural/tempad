# T-P406: Task selection flow — claim → workspace → IDE

**Ticket:** T-P406 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 9.3 | **Deps:** T-P400, T-P302, T-P405

## Description
Full flow when developer presses Enter: claim → prepare workspace → open IDE.

## Files
- `internal/tui/app.go` (expand Update)

## Work Items
- [ ] Enter → "Claiming..." → claim.Claim() → ClaimResultMsg
- [ ] Success → prepareWorkspaceCmd → WorkspaceReadyMsg
- [ ] Workspace ready → `bash -lc "<ide.command> <ide.args> <path>"` → IDEOpenedMsg
- [ ] "Opened in IDE" status → return to board
- [ ] Claim failure → error message, return to board
- [ ] Disable selection while claim in progress

## Acceptance Criteria
- [ ] Full flow: select → claim → workspace → IDE opens
- [ ] Race lost → "Someone else claimed this task"
- [ ] Hook failure → error, board still usable
- [ ] Can pick another task after IDE opens

## Research Notes (2026-03-08)

- The claim → workspace → IDE flow should use chained Bubble Tea commands, not blocking goroutines. Each step returns a Cmd that triggers the next step via a message.
- For IDE launch: `exec.Command("bash", "-lc", fmt.Sprintf("%s %s %s", ideCmd, ideArgs, path))` — must not block the TUI event loop.
- Use `tea.ExecProcess` for launching the IDE if it needs terminal control (e.g., Vim), otherwise spawn in background.
