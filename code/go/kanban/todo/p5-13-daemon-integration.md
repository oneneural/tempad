# T-P513: Daemon integration test

**Ticket:** T-P513 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 20.1 | **Deps:** T-P512

## Description

Mock tracker (3 issues), mock agent (echo done). Verify: claimed, workers, continuation retry, terminal cleanup, max retries release. go test -race, goleak

## Files

- `internal/orchestrator/integration_test.go`

## Work Items

- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria

- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- Use `httptest.NewServer()` for the mock tracker — return canned GraphQL responses for each query type.
- Mock agent: `echo "done"` with configurable exit code and output delay.
- Use `goleak.VerifyNone(t)` in `TestMain` to catch goroutine leaks.
- Use `t.TempDir()` for workspace directories — auto-cleanup.
- Test the full lifecycle: poll → select → claim → dispatch → worker complete → continuation retry → terminal cleanup → max retries release.
- Consider `testify/suite` for organizing the integration test phases.
