# T-P800: Unit test coverage for all packages

**Ticket:** T-P800 | **Phase:** 8 — Testing + Hardening | **Status:** 🔲 TODO
**Spec:** Section 20.1 | **Deps:** All phases

## Description
Fill test coverage gaps across all packages. Focus: config merge (10+ cases), workflow edges, sanitization, candidate selection, backoff formula, concurrency, hooks, prompts.

## Acceptance Criteria
- [ ] `go test ./...` passes
- [ ] All Spec 20.1 test cases covered

## Research Notes (2026-03-08)

- Use `testify/assert` + `testify/require` for cleaner assertions.
- Table-driven tests for: config merge (10+), candidate selection (8+), backoff formula (5+), sanitization (5+).
- Mock Linear API with `httptest.NewServer()`. Use `t.TempDir()` for filesystem tests.
- Target coverage: config, workflow, sanitization, candidate selection, backoff, hooks, prompts.
