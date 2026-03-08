# T-P601: Structured logging setup

**Ticket:** T-P601 | **Phase:** 6 — Hot Reload + Logging | **Status:** 🔲 TODO
**Spec:** Section 15.1, 15.2 | **Deps:** T-P105

## Description
Configure slog with mode-specific sinks, log levels, rotation, and per-issue agent logs.

## Files
- `internal/logging/setup.go`
- `internal/logging/rotate.go`

## Work Items
- [ ] TUI → stderr sink. Daemon → file sink (`~/.tempad/logs/tempad.log`)
- [ ] Log level from config, key=value format
- [ ] Context fields: issue_id, issue_identifier, mode, attempt, agent_pid
- [ ] Rotation: 50MB, keep 5. Create dirs automatically.
- [ ] Per-issue: `~/.tempad/logs/<identifier>/agent.log`

## Acceptance Criteria
- [ ] TUI logs to stderr, daemon to file
- [ ] Level filtering works
- [ ] Rotation triggers at size limit

## Research Notes (2026-03-08)

- **slog has no built-in rotation** — must use `lumberjack` (natefinish/lumberjack) as the `io.Writer` for the slog handler.
- **Lumberjack config**: `MaxSize: 50` (MB), `MaxBackups: 5`, `MaxAge: 30` (days), `Compress: true`.
- Wire it as: `slog.New(slog.NewJSONHandler(&lumberjack.Logger{...}, nil))` for daemon, `slog.New(slog.NewTextHandler(os.Stderr, nil))` for TUI.
- Use `slog.With("mode", "daemon")` at setup to tag all log entries with the running mode.
- Per-issue log files should also use lumberjack for rotation, but with smaller limits (10MB, keep 3).
