# T-P103: User config loader

**Ticket:** T-P103
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 7.1, 7.2, 7.3

## Description

Implement `LoadUserConfig(path)` for `~/.tempad/config.yaml` with nested struct types.

## Files Created

- `internal/config/user.go` — UserConfig, nested structs, LoadUserConfig, DefaultUserConfigPath, DefaultUserConfigTemplate
- `internal/config/user_test.go` — 4 tests

## Acceptance Criteria

- [x] Loads config correctly
- [x] Missing file → empty config (not error)
- [x] Malformed YAML → parse error
- [x] `$VAR` values preserved as-is
- [x] 4 unit tests pass

## Dependencies

T-P100
