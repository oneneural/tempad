# Phase 8: Testing + Hardening

**Status:** 🔲 PENDING
**Tickets:** T-P800 to T-P805 (6 tickets)
**Prerequisites:** All previous phases
**Goal:** Full test coverage, race detection, goroutine leak prevention, production readiness.

## Success Criteria

- [ ] `go test ./...` passes across all packages
- [ ] `go test -race ./...` passes with zero warnings
- [ ] goleak reports zero goroutine leaks
- [ ] End-to-end integration test passes (mock tracker + mock agent)
- [ ] SIGINT/SIGTERM trigger clean shutdown (exit 0, all claims released)
- [ ] Real Linear smoke test passes
- [ ] No orphaned assignments after tests
- [ ] All Spec Section 20.1 test cases covered

## Task List

| # | Ticket | Task | Status | File | Deps |
|---|--------|------|--------|------|------|
| 1 | T-P800 | Unit test coverage for all packages | 🔲 Todo | `p8-00-unit-tests.md` | All phases |
| 2 | T-P801 | Race condition detection | 🔲 Todo | `p8-01-race-detection.md` | T-P800 |
| 3 | T-P802 | Goroutine leak detection | 🔲 Todo | `p8-02-goroutine-leaks.md` | T-P800 |
| 4 | T-P803 | End-to-end integration test | 🔲 Todo | `p8-03-e2e-test.md` | All phases |
| 5 | T-P804 | Signal handling verification | 🔲 Todo | `p8-04-signal-handling.md` | T-P512 |
| 6 | T-P805 | Real Linear smoke test | 🔲 Todo | `p8-05-linear-smoke.md` | All phases |

## Dependency Order

```
{T-P800} → {T-P801, T-P802} → {T-P803, T-P804, T-P805}
```

T-P800 first (fill gaps). Then T-P801/T-P802 in parallel. Then T-P803/T-P804/T-P805 in parallel.

## Research Findings (2026-03-08)

**Validated:**
- `goleak.VerifyNone(t)` for goroutine leak detection — use in `TestMain` or per-test.
- `go test -race ./...` with extended timeout for integration tests.

**Recommendations:**
- Use `testify/assert` + `testify/require` for cleaner assertions.
- Mock Linear with `httptest.NewServer()` returning canned GraphQL responses.
- Use `t.TempDir()` for workspace tests (auto-cleanup).
- E2E: mock tracker + mock agent subprocess, verify full lifecycle.
- Add `go vet ./...` and `staticcheck ./...` to CI pipeline.
- Smoke tests tagged `//go:build smoke`, skipped when env vars absent.

See `handoffs/research-findings.md` for full details.
