# T-P105: ServiceConfig struct and merge logic

**Ticket:** T-P105
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 8.1, 8.4, 4.1.4

## Description

Define ServiceConfig (33 fields), CLIFlags, Defaults(), and Merge with 5-level precedence.

## Files Created

- `internal/config/config.go` — ServiceConfig (33 fields), CLIFlags, Defaults()
- `internal/config/loader.go` — Merge, applyWorkflowConfig, applyUserConfig, applyCLIFlags, Load pipeline
- `internal/config/loader_test.go` — 10 tests

## Acceptance Criteria

- [x] CLI `--identity` overrides user config
- [x] User `agent.command` overrides repo
- [x] Repo `hooks.after_create` NOT overridable by user
- [x] All defaults applied when configs empty
- [x] Comma-separated states parsed to list
- [x] 10+ unit tests pass

## Dependencies

T-P102, T-P103, T-P104
