# T-P801: Race condition detection

**Ticket:** T-P801 | **Phase:** 8 — Testing + Hardening | **Status:** 🔲 TODO
**Spec:** Architecture Section 13 | **Deps:** T-P800

## Description

Run `go test -race ./...`, fix any races. Verify channel-only communication in orchestrator.

## Acceptance Criteria

- [ ] `go test -race ./...` passes with zero warnings

## Research Notes (2026-03-08)

- Common race sources: orchestrator state maps, lastOutputAt atomic, retry timer callbacks, config reload channel.
- Race detector adds ~10x overhead — use `-timeout 300s` for integration tests.
- Verify channel-only communication in orchestrator — no shared mutable state except through channels or atomics.
