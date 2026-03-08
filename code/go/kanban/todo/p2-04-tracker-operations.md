# T-P204: Implement all 6 tracker operations

**Ticket:** T-P204
**Phase:** 2 — Tracker Client (Linear)
**Status:** 🔲 TODO
**Spec:** Section 13.1
**Deps:** T-P202, T-P203

## Problem

Need all 6 tracker operations implemented against the Linear GraphQL API.

## Solution

Implement each method on `LinearClient` using the queries from T-P201 and normalization from T-P203.

## Files

- `internal/tracker/linear/client.go` (expand)

## Work Items

- [ ] `FetchCandidateIssues(ctx)` — unassigned + active states + my assigned (resumption). Merge and deduplicate.
- [ ] `FetchIssuesByStates(ctx, states)` — terminal cleanup query
- [ ] `FetchIssueStatesByIDs(ctx, ids)` — batch node lookup → `map[id]state`
- [ ] `FetchIssue(ctx, id)` — single issue fetch
- [ ] `AssignIssue(ctx, issueID, identity)` — mutation
- [ ] `UnassignIssue(ctx, issueID)` — mutation with `assigneeId: null`
- [ ] Identity resolution: if email → resolve to Linear user ID at construction (cache)

## Acceptance Criteria

- [ ] Each operation handles success and error cases
- [ ] `FetchCandidateIssues` returns normalized, deduplicated issues
- [ ] Identity resolution from email works
- [ ] 6 unit tests (one per operation) with mock server

## Research Notes (2026-03-08)

- **Identity resolution**: Use `users(filter: { email: { eq: "..." } })` query — cache result at client construction time.
- **Claim verification**: After `AssignIssue`, immediately `FetchIssue` to verify the assignee matches — this is the optimistic concurrency check.
- **GraphQL 200 + errors**: All operations must check for `errors[]` in response even on HTTP 200.
- Consider using `context.WithTimeout` per-operation (30s default) on top of the parent context.
