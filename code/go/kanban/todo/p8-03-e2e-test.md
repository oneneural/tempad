# T-P803: End-to-end integration test

**Ticket:** T-P803 | **Phase:** 8 — Testing + Hardening | **Status:** 🔲 TODO
**Spec:** Section 20.1, 20.3 | **Deps:** All phases

## Description

Full e2e: temp WORKFLOW.md + mock tracker + mock agent. Verify full lifecycle, logs, workspaces, config reload, shutdown.

## Files

- `test/e2e_test.go`

## Acceptance Criteria

- [ ] Full lifecycle passes
- [ ] Logs contain expected entries
- [ ] Workspaces managed correctly
- [ ] No races or leaks

## Research Notes (2026-03-08)

- E2E test structure: `TestMain` sets up temp dir + mock tracker + mock agent binary → run orchestrator with short poll interval → verify lifecycle events → clean shutdown.
- Mock agent: compile a small Go binary that reads PROMPT.md, echoes output, exits with configurable code (via env var).
- Use `os.Setenv("TEMPAD_TEST_AGENT_EXIT_CODE", "0")` to control mock agent behavior per test case.
- Verify: claim happened, workspace created, agent received correct env vars, continuation retry works, terminal cleanup works.
