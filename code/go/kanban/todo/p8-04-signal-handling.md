# T-P804: Signal handling and graceful shutdown verification

**Ticket:** T-P804 | **Phase:** 8 — Testing + Hardening | **Status:** 🔲 TODO
**Spec:** Section 18.2 | **Deps:** T-P512

## Description
Verify SIGINT/SIGTERM trigger clean shutdown: exit 0, all agents terminated, all claims released, within 30s.

## Files
- `cmd/tempad/main_test.go`

## Acceptance Criteria
- [ ] Both signals → exit 0
- [ ] No orphan processes
- [ ] Shutdown within 30s

## Research Notes (2026-03-08)

- Test signal handling by spawning the daemon as a subprocess, sending SIGTERM via `syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)`, then verifying clean exit.
- Verify that after shutdown: no orphan agent processes, all claimed issues are unassigned (check via mock tracker), exit code is 0.
- Use `cmd.ProcessState.ExitCode()` to verify the exit code after `cmd.Wait()`.
- Test both SIGINT and SIGTERM — they should behave identically.
