# T-P301: Hook execution engine

**Ticket:** T-P301
**Phase:** 3 — Workspace Manager + Hooks
**Status:** 🔲 TODO
**Spec:** Section 12.4, 6.3.4
**Deps:** T-P100

## Problem

Workspace lifecycle hooks need to run shell scripts in the workspace directory with timeout enforcement.

## Solution

`RunHook(ctx, name, script, workspaceDir, timeoutMs)` that executes via `bash -lc` with process group kill on timeout.

## Files

- `internal/workspace/hooks.go`

## Work Items

- [ ] `RunHook(ctx, name, script, workspaceDir, timeoutMs) error`
- [ ] Execute via `bash -lc <script>` with cwd = workspace
- [ ] Timeout via `context.WithTimeout`, kill process group (not just parent)
- [ ] Capture stdout/stderr for logging (truncate in logs)
- [ ] Return error on non-zero exit or timeout
- [ ] Log hook start, completion, failure, timeout

## Acceptance Criteria

- [ ] `echo hello` runs and succeeds
- [ ] `exit 1` returns error
- [ ] `sleep 999` with 100ms timeout → killed, timeout error
- [ ] Script runs with correct cwd
- [ ] Stdout/stderr captured
- [ ] Unit tests pass

## Research Notes (2026-03-08)

- **Process group**: Set `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` to put hook process in its own process group.
- **Kill entire group**: Use `syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)` (negative PID) to kill all child processes, not just the lead.
- **Timeout pattern**: Use `context.WithTimeout` + goroutine that waits on `ctx.Done()`, then sends SIGTERM → 5s grace → SIGKILL escalation.
- **Environment variables**: Set `TEMPAD_ISSUE_ID`, `TEMPAD_WORKSPACE_DIR` in the hook's env for custom scripts.
- Log hook stdout/stderr via `slog` at debug level for troubleshooting.
- Use `cmd.StdoutPipe()` / `cmd.StderrPipe()` with `io.Copy` to a buffer, truncate at ~4KB for log output.
