# T-P507: Agent subprocess launcher

**Ticket:** T-P507 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 11.1, 11.3-4 | **Deps:** T-P506

## Description
SubprocessLauncher: bash -lc command, set cwd/env, handle prompt delivery, Return RunHandle with Wait/Cancel/Stdout/Stderr. Cancel: SIGTERM → 5s → SIGKILL

## Files
- `internal/agent/launcher.go, process.go`

## Work Items
- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria
- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- **CRITICAL**: Set `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` for process group isolation.
- **Kill escalation**: SIGTERM → 5s → `syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)` (negative PID for process group).
- Use `context.WithCancel` + goroutine watching `ctx.Done()` for kill escalation.
- Go 1.20+: `exec.Cmd` has `WaitDelay` and `Cancel` fields for customizable context cancellation behavior.
