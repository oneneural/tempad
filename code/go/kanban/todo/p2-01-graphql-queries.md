# T-P201: Linear GraphQL query/mutation builders

**Ticket:** T-P201
**Phase:** 2 — Tracker Client (Linear)
**Status:** 🔲 TODO
**Spec:** Section 13.2, Architecture Section 8.1
**Deps:** T-P100

## Problem

Need GraphQL queries and mutations to interact with Linear's API for all 6 tracker operations.

## Solution

Define GraphQL query strings as Go constants with request/response structs for JSON marshaling.

## Files

- `internal/tracker/linear/graphql.go`

## Work Items

- [ ] `candidateIssuesQuery` — filter by project slug, active states, unassigned. Pagination. All Issue fields (id, identifier, title, description, priority, state.name, assignee.id/email, branchName, url, labels.nodes.name, relations for blockers, createdAt, updatedAt)
- [ ] `assignedToMeQuery` — same but filter by current user (resumption)
- [ ] `issueStatesByIDsQuery` — batch node lookup, return id + state.name
- [ ] `issuesByStatesQuery` — filter by state names (terminal cleanup)
- [ ] `singleIssueQuery` — one issue by ID (claim verification)
- [ ] `assignIssueMutation` — `issueUpdate(id, input: { assigneeId })`
- [ ] `unassignIssueMutation` — `issueUpdate(id, input: { assigneeId: null })`
- [ ] Request/response structs for JSON marshaling
- [ ] GraphQL error response parsing

## Acceptance Criteria

- [ ] All query strings are valid GraphQL
- [ ] Response structs unmarshal real Linear responses
- [ ] GraphQL error format `{ errors: [{ message }] }` detected
- [ ] Compile check passes

## Research Notes (2026-03-08)

- **CRITICAL**: Use `project.slug` (not `slugId`) when filtering issues by project. `slugId` is deprecated in the Linear API.
- **Identity resolution query**: `users(filter: { email: { eq: "..." } })` returns user ID from email — add this to the query set.
- **Relation direction**: Linear's `issueRelations` returns `type` and `relatedIssue`. For blockers, look for relations where `type == "blocks"` on the inverse side.
- Add `X-Request-Id` header generation for debugging/tracing.
- Consider `@genqlient` or `shurcooL/graphql` for type-safe query generation instead of raw string constants.

## Parallelizable

Yes — can be done in parallel with T-P200.
