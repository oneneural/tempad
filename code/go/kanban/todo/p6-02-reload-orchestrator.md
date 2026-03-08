# T-P602: Config reload integration with orchestrator

**Ticket:** T-P602 | **Phase:** 6 — Hot Reload + Logging | **Status:** 🔲 TODO
**Spec:** Section 8.2 | **Deps:** T-P600, T-P501

## Description
Handle configReload channel in orchestrator select loop. Update poll interval, max_concurrent, backoff/timeout, prompt template. Don't restart in-flight agents.

## Files
- `internal/orchestrator/orchestrator.go` (configReload case)

## Work Items
- [ ] Update poll_interval → reset ticker
- [ ] Update max_concurrent, backoff/timeout, prompt template
- [ ] Log which fields changed
- [ ] Do NOT restart in-flight agents

## Acceptance Criteria
- [ ] poll_interval change → tick interval changes
- [ ] max_concurrent change → applied next dispatch
- [ ] In-flight agents unaffected

## Research Notes (2026-03-08)

- Resetting the ticker on poll_interval change: `ticker.Reset(newInterval)` (Go 1.15+).
- For prompt template changes, the new template only applies to the next dispatch cycle — never re-render prompts for in-flight agents.
- Log a structured diff of which config fields changed (old vs new values) for operational debugging.
