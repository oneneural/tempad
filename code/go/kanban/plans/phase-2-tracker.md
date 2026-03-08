# Phase 2: Tracker Client (Linear)

**Status:** 🔲 PENDING
**Tickets:** T-P200 to T-P205 (6 tickets)
**Prerequisites:** Phase 1 (complete)
**Goal:** All 6 tracker operations work against Linear's GraphQL API. Issues normalized into domain model.

## Success Criteria

- [ ] `tracker.Client` interface with 6 operations compiles
- [ ] All typed errors implement `error` and support `errors.Is`/`errors.As`
- [ ] GraphQL queries fetch correct data from Linear API
- [ ] Cursor-based pagination works (tested with 3-page mock)
- [ ] Issue normalization: labels lowercase, priorities as int, blockers resolved
- [ ] Claim flow: assign → fetch → verify works
- [ ] Identity resolution from email to Linear user ID works
- [ ] Integration smoke test passes against real Linear API

## Task List

| # | Ticket | Task | Status | File | Deps |
| --- | -------- | ------ | -------- | ------ | ------ |
| 1 | T-P200 | Tracker client interface and error types | 🔲 Todo | `p2-00-tracker-interface.md` | T-P101 |
| 2 | T-P201 | Linear GraphQL query/mutation builders | 🔲 Todo | `p2-01-graphql-queries.md` | T-P100 |
| 3 | T-P202 | Linear HTTP transport and pagination | 🔲 Todo | `p2-02-http-transport.md` | T-P200, T-P201 |
| 4 | T-P203 | Issue normalization (Linear → domain) | 🔲 Todo | `p2-03-issue-normalization.md` | T-P201, T-P101 |
| 5 | T-P204 | Implement all 6 tracker operations | 🔲 Todo | `p2-04-tracker-operations.md` | T-P202, T-P203 |
| 6 | T-P205 | Tracker integration smoke test | 🔲 Todo | `p2-05-tracker-smoke-test.md` | T-P204 |

## Dependency Order

```text
{T-P200, T-P201} → T-P202 → T-P203 → T-P204 → T-P205
```

T-P200 and T-P201 can be done in parallel. T-P202 needs both. T-P203 needs T-P201. T-P204 needs T-P202 and T-P203.

## Parallelization with Phase 3

Phase 2 and Phase 3 can be developed in parallel — they have no cross-dependencies until Phase 4 (TUI) needs both.

## Research Findings (2026-03-08)

**Key corrections:**

- **CRITICAL**: Use `project.slug` (not `slugId`) when filtering issues — `slugId` is deprecated in the Linear API.
- Linear returns HTTP 200 with `errors[]` for GraphQL failures — must check response body even on 200.
- Rate limit: 5,000 requests/hour per API key (complexity-weighted).

**Recommendations:**

- Consider `@genqlient` or `shurcooL/graphql` for type-safe GraphQL client.
- Cache identity resolution (email → user ID) at client construction.
- Add `X-Request-Id` header for debugging/tracing.

See `handoffs/research-findings.md` for full details.
