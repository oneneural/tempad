# T-P101: Define all domain model structs

**Ticket:** T-P101
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 4.1, 4.2

## Description

Define all core domain structs: Issue (14 fields), BlockerRef, Workspace, RunAttempt, RetryEntry, OrchestratorState, AgentTotals. Implement SanitizeIdentifier and NormalizeState utilities.

## Files Created

- `internal/domain/issue.go` — Issue struct, BlockerRef, HasNonTerminalBlockers
- `internal/domain/workspace.go` — Workspace struct
- `internal/domain/run.go` — RunAttempt, RetryEntry, AgentTotals
- `internal/domain/state.go` — OrchestratorState with NewOrchestratorState, Snapshot, RunningCount, IsClaimedOrRunning, AvailableSlots
- `internal/domain/normalize.go` — SanitizeIdentifier, NormalizeState, NormalizeStates
- `internal/domain/normalize_test.go` — Unit tests

## Acceptance Criteria

- [x] All structs compile
- [x] `SanitizeIdentifier("ABC-123/foo bar")` returns `"ABC-123_foo_bar"`
- [x] `NormalizeState("  In Progress  ")` returns `"in progress"`
- [x] Unit tests pass

## Dependencies

T-P100
