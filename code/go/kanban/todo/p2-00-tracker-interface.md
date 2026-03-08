# T-P200: Tracker client interface and error types

**Ticket:** T-P200
**Phase:** 2 — Tracker Client (Linear)
**Status:** 🔲 TODO
**Spec:** Section 13.1, 13.4
**Deps:** T-P101

## Problem

Need a clean interface for tracker operations so Linear is swappable and test doubles work. Need typed errors for proper error handling throughout the codebase.

## Solution

Define `tracker.Client` interface with 6 methods and a comprehensive set of typed errors.

## Files

- `internal/tracker/client.go` — Client interface (already stubbed)
- `internal/tracker/errors.go` — Typed errors (already stubbed)

## Work Items

- [ ] Define `Client` interface with 6 methods:
  - `FetchCandidateIssues(ctx) ([]domain.Issue, error)`
  - `FetchIssueStatesByIDs(ctx, ids) (map[string]string, error)`
  - `FetchIssuesByStates(ctx, states) ([]domain.Issue, error)`
  - `FetchIssue(ctx, id) (*domain.Issue, error)`
  - `AssignIssue(ctx, issueID, identity) error`
  - `UnassignIssue(ctx, issueID) error`
- [ ] Define typed errors:
  - `UnsupportedTrackerKindError`
  - `MissingTrackerAPIKeyError`
  - `MissingTrackerProjectSlugError`
  - `MissingTrackerIdentityError`
  - `TrackerAPIRequestError` (transport)
  - `TrackerAPIStatusError` (non-200)
  - `TrackerAPIErrorsError` (GraphQL errors)
  - `TrackerClaimConflictError` (assignment race)
- [ ] All errors implement Go `error` interface
- [ ] `errors.Is` / `errors.As` work correctly

## Acceptance Criteria

- [ ] Interface compiles
- [ ] Errors implement `error` with structured messages (issue ID, HTTP status, etc.)
- [ ] `errors.Is` / `errors.As` work
- [ ] Unit tests for error creation and matching

## Research Notes (2026-03-08)

- **GraphQL 200 + errors**: Linear returns HTTP 200 with `errors[]` array for GraphQL failures. `TrackerAPIErrorsError` must be checked even on 200 status — don't assume 200 = success.
- **Rate limiting**: Linear allows 5,000 requests/hour per API key (complexity-weighted). Consider adding a `TrackerRateLimitError` type for 429 responses.
- Consider using `@genqlient` or `shurcooL/graphql` for type-safe GraphQL client generation.

## Parallelizable

Yes — can be done in parallel with T-P201.
