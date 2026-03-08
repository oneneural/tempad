# T-P106: Dispatch preflight validation

**Ticket:** T-P106
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 8.3

## Description

Implement ValidateForStartup and ValidateForDispatch with 6 validation checks.

## Files Created

- `internal/config/validation.go` — ValidationError, ValidationErrors, ValidateForStartup, ValidateForDispatch
- `internal/config/validation_test.go` — 8 tests

## Acceptance Criteria

- [x] Missing tracker.kind → clear error
- [x] Missing agent.command in daemon → error
- [x] Missing agent.command in TUI → no error
- [x] Empty api_key after $VAR → "resolved to empty" error
- [x] All 6 checks tested
- [x] 8 unit tests pass

## Dependencies

T-P105
