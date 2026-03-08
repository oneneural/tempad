# T-P400: Claim mechanism (shared by TUI + daemon)

**Ticket:** T-P400 | **Phase:** 4 — TUI Mode | **Status:** 🔲 TODO
**Spec:** Section 5.1, 5.2, 5.3 | **Deps:** T-P200

## Description
Stateless claim/release operations: assign → fetch → verify (race detection) → release on conflict.

## Files
- `internal/claim/claimer.go`

## Work Items
- [ ] `Claim(ctx, tracker, issueID, identity)`: assign → fetch → verify assignee → unassign + ClaimConflictError if mismatch
- [ ] `Release(ctx, tracker, issueID)`: unassign
- [ ] Stateless — all state managed by caller

## Acceptance Criteria
- [ ] Successful claim assigns and verifies
- [ ] Race lost → unassigns, returns conflict error
- [ ] Tracker error in step 1 → returns error without step 2
- [ ] Unit tests with mock tracker

## Research Notes (2026-03-08)

- The claim mechanism uses Linear's `issueUpdate` mutation as a distributed lock with optimistic concurrency — no external lock service needed.
- **Claim verification**: After `AssignIssue`, immediately `FetchIssue` and check if `assignee.id` matches — this handles the race window where two agents assign simultaneously (Linear uses last-write-wins).
- Consider adding a small configurable delay (50-100ms) between assign and verify to allow Linear's eventual consistency to settle.
