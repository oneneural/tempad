# T-P107: Cobra CLI with init and validate commands

**Ticket:** T-P107
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 18.1, 18.2

## Description

Root command with all flags, `tempad init` and `tempad validate` subcommands.

## Files Created

- `cmd/tempad/main.go` — Root command with --daemon, --workflow, --identity, --agent, --ide, --port, --log-level
- `cmd/tempad/init.go` — Creates ~/.tempad/config.yaml
- `cmd/tempad/validate.go` — Loads, merges, validates, prints result
- `cmd/tempad/clean.go` — Placeholder for Phase 3

## Acceptance Criteria

- [x] `tempad init` creates config, won't overwrite existing
- [x] `tempad validate` → exit 0 (valid) or exit 1 (errors)
- [x] All flags parsed into CLIFlags struct

## Dependencies

T-P105, T-P106
