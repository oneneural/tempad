# T-P104: Environment variable resolution

**Ticket:** T-P104
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 8.1 (item 4), Section 6.3.1

## Description

Implement `ResolveEnvVar` ($VAR → os.Getenv) and `ExpandHome` (~/path expansion).

## Files Created

- `internal/config/resolve.go` — ResolveEnvVar, ExpandHome
- `internal/config/resolve_test.go` — Tests with controlled env vars

## Acceptance Criteria

- [x] `$LINEAR_API_KEY` resolves when env var set
- [x] `$NONEXISTENT` resolves to "" (treated as missing)
- [x] `~/workspaces` expands to absolute home path
- [x] Unit tests pass

## Dependencies

T-P100
