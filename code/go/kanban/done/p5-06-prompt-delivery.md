# T-P506: Prompt delivery (4 methods)

**Ticket:** T-P506 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 11.3, 6.3.5 | **Deps:** T-P100

## Description

DeliverPrompt: file (PROMPT.md), stdin (io.Reader), arg (CLI argument), env (TEMPAD_PROMPT). Returns DeliveryResult with stdinPipe, extraArgs, extraEnv

## Files

- `internal/agent/delivery.go`

## Work Items

- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria

- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- **File delivery**: Write `PROMPT.md` to workspace dir. Simplest and most debuggable method.
- **Stdin delivery**: Use `cmd.StdinPipe()` → write → close. Must close the pipe or the agent will hang waiting for EOF.
- The prompt delivery method should be configurable per-workflow, not hard-coded.
