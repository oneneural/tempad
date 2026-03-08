# T-P805: Real Linear smoke test suite

**Ticket:** T-P805 | **Phase:** 8 — Testing + Hardening | **Status:** 🔲 TODO
**Spec:** Section 20.3 | **Deps:** All phases

## Description
Tagged `//go:build smoke`. Real Linear credentials. Fetch, claim/release, normalize, run echo agent, cleanup.

## Files
- `test/smoke_test.go`

## Acceptance Criteria
- [ ] Passes against real Linear API
- [ ] No orphaned assignments
- [ ] Cleanup always runs

## Research Notes (2026-03-08)

- Tag with `//go:build smoke` — run separately from unit tests via `go test -tags smoke ./test/...`.
- Requires `LINEAR_API_KEY` and `LINEAR_TEST_PROJECT_SLUG` env vars — skip gracefully if absent.
- Use `t.Cleanup()` for unassign/cleanup steps to ensure they run even on test failure.
- Rate limit: tests count against the 5,000 req/hr limit. Add small delays between operations.
- Verify `project.slug` (not `slugId`) is used throughout the test fixtures.
