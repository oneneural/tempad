# Phase 6: Hot Reload + Logging + Polish

**Status:** ✅ COMPLETE
**Tickets:** T-P600 to T-P603 (4 tickets)
**Prerequisites:** Phase 4 (TUI), Phase 5 (Daemon)
**Goal:** Dynamic config reload, structured logging, production-ready polish.

## Success Criteria

- [ ] Edit WORKFLOW.md → new config applied within 1s
- [ ] Rapid edits debounced to single reload
- [ ] Invalid reload keeps last known good config
- [ ] TUI mode logs to stderr (doesn't interfere with TUI)
- [ ] Daemon mode logs to file with rotation (50MB, 5 files)
- [ ] Per-issue agent logs at `~/.tempad/logs/<identifier>/agent.log`
- [ ] Config change in orchestrator updates poll interval, max_concurrent, etc.
- [ ] In-flight agents not restarted on config change
- [ ] TUI shows "Config reloaded" status message

## Task List

| # | Ticket | Task | Status | File | Deps |
| --- | -------- | ------ | -------- | ------ | ------ |
| 1 | T-P600 | WORKFLOW.md file watcher + debounce | ✅ Done | `p6-00-file-watcher.md` | T-P105 |
| 2 | T-P601 | Structured logging setup | ✅ Done | `p6-01-logging-setup.md` | T-P105 |
| 3 | T-P602 | Config reload → orchestrator | ✅ Done | `p6-02-reload-orchestrator.md` | T-P600, T-P501 |
| 4 | T-P603 | Config reload → TUI | ✅ Done | `p6-03-reload-tui.md` | T-P600, T-P401 |

## Dependency Order

```text
{T-P600, T-P601} → {T-P602, T-P603}
```

T-P600 and T-P601 can be done in parallel. T-P602 and T-P603 can also be done in parallel (they integrate with different modes).

## Research Findings (2026-03-08)

**Key corrections:**

- **CRITICAL**: fsnotify must watch the **directory**, not the file. Vim/Emacs use rename-and-replace which loses file watches.
- **slog has no built-in log rotation** — must use `lumberjack` (natefinish/lumberjack) as the io.Writer.

**Recommendations:**

- Lumberjack config: `MaxSize: 50`, `MaxBackups: 5`, `MaxAge: 30`, `Compress: true`.
- Debounce pattern: `time.AfterFunc` with reset on each event.
- TUI config reload via `tea.Cmd` that reads from channel.

See `handoffs/research-findings.md` for full details.
