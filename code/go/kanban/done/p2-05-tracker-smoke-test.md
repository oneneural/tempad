# T-P205: Tracker client integration smoke test

**Ticket:** T-P205
**Phase:** 2 — Tracker Client (Linear)
**Status:** 🔲 TODO
**Spec:** Section 20.3
**Deps:** T-P204

## Problem

Need to verify the tracker client works against real Linear API.

## Solution

Integration test tagged `//go:build integration` that exercises real API calls.

## Files

- `internal/tracker/linear/integration_test.go`

## Work Items

- [ ] Test tagged `//go:build integration`
- [ ] Requires `LINEAR_API_KEY` and `LINEAR_TEST_PROJECT_SLUG` env vars
- [ ] Fetch candidates from real Linear project
- [ ] Verify normalization on real data
- [ ] Assign/unassign a designated test issue
- [ ] Clean up after test (always, even on failure)

## Acceptance Criteria

- [ ] Passes against real Linear API
- [ ] No orphaned assignments
- [ ] Skips gracefully when env vars absent

## Research Notes (2026-03-08)

- **Rate limit awareness**: Integration tests count against the 5,000 req/hr limit. Add small delays between operations or use a dedicated test API key.
- Use `t.Cleanup()` for the unassign step to ensure it runs even on test failure.
- Verify that `project.slug` (not `slugId`) is used in the test configuration.
- Consider a `TestMain` that validates env vars and skips the entire file if missing.
