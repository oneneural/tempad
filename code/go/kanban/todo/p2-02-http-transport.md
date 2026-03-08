# T-P202: Linear HTTP transport and pagination

**Ticket:** T-P202
**Phase:** 2 — Tracker Client (Linear)
**Status:** 🔲 TODO
**Spec:** Section 13.2
**Deps:** T-P200, T-P201

## Problem

Need HTTP transport to send GraphQL queries to Linear with proper auth, error handling, and cursor-based pagination.

## Solution

`LinearClient` struct with `do()` method for GraphQL requests and `fetchAll()` helper for pagination.

## Files

- `internal/tracker/linear/client.go`
- `internal/tracker/linear/pagination.go`

## Work Items

- [ ] `LinearClient` struct: httpClient, endpoint, apiKey, projectSlug, identity, timeout (30s)
- [ ] `NewLinearClient(cfg)` constructor
- [ ] `do(ctx, query, vars, result)` — POST with `Authorization: Bearer <key>`, JSON body, unmarshal, check errors
- [ ] Cursor-based pagination: `fetchAll[T](ctx, query, vars, extractPage)` looping until `hasNextPage` false
- [ ] Default page size: 50
- [ ] HTTP timeout: 30s
- [ ] Respect `context.Context` cancellation

## Acceptance Criteria

- [ ] Sends correct `Authorization` header
- [ ] Paginates correctly (mock server with 3 pages)
- [ ] Context cancellation aborts in-flight request
- [ ] Non-200 → `TrackerAPIStatusError`
- [ ] Network error → `TrackerAPIRequestError`
- [ ] Unit tests with httptest mock server

## Research Notes (2026-03-08)

- **CRITICAL**: Linear returns HTTP 200 with `errors[]` for GraphQL failures — must parse response body and check for `errors` field even on 200 status.
- **Rate limit**: 5,000 req/hr (complexity-weighted). Response may include `X-RateLimit-*` headers. Add retry-after logic for 429 responses.
- **CRITICAL**: Use `project.slug` (not `slugId`) for project filter in queries. `slugId` is deprecated.
- Add `X-Request-Id` header (UUID per request) for debugging/tracing.
- Cache user identity resolution (email → user ID) — it won't change during a session.
