# T-P512: Daemon mode entry point

**Ticket:** T-P512 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 10.1, 18.1-2 | **Deps:** T-P501–T-P511

## Description
--daemon flag: load+merge+validate (agent.command required) → create components → orchestrator.Run(ctx). SIGINT/SIGTERM → cancel

## Files
- `cmd/tempad/main.go`

## Work Items
- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria
- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- Use `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` for clean signal handling.
- Validate `agent.command` is non-empty during preflight — this is a daemon-only requirement.
- Log the daemon startup config (redact API key) for operational debugging.
- Consider writing a PID file to prevent multiple daemon instances on the same project.
