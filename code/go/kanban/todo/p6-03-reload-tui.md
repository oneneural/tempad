# T-P603: Config reload integration with TUI

**Ticket:** T-P603 | **Phase:** 6 — Hot Reload + Logging | **Status:** 🔲 TODO
**Spec:** Section 8.2 | **Deps:** T-P600, T-P401

## Description

Handle ConfigReloadMsg in TUI model: update config, reset poll interval, show status message.

## Files

- `internal/tui/app.go` (ConfigReloadMsg)

## Work Items

- [ ] Update internal config reference
- [ ] Reset poll interval timer
- [ ] Show "Config reloaded" status
- [ ] Invalid → error in status bar

## Acceptance Criteria

- [ ] Config change reflected
- [ ] Poll interval change takes effect
- [ ] Error shown for invalid config

## Research Notes (2026-03-08)

- The watcher sends config on a channel — the TUI should receive it as a `ConfigReloadMsg` via a `tea.Cmd` that reads from the channel.
- Pattern: `func waitForReload(ch <-chan config.ServiceConfig) tea.Cmd { return func() tea.Msg { return ConfigReloadMsg(<-ch) } }` — called in Init and re-called after each receive.
