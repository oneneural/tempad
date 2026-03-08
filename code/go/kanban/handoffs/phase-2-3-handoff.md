# Phase 2+3 Handoff: Tracker Client & Workspace Manager Complete

**Date:** 2026-03-08
**Status:** Phases 1, 2, 3 COMPLETE (21/57 tickets). Ready for Phase 4+5 development.

---

## What Was Built

### Phase 2: Linear GraphQL Tracker Client (6 tickets)

Full `tracker.Client` implementation for Linear's GraphQL API.

| Ticket | What It Does |
| --- | --- |
| T-P200 | Verified `tracker.Client` interface (6 methods), added `RateLimitError`, unit tests for all 9 error types |
| T-P201 | 8 GraphQL query/mutation constants, shared `IssueFields` fragment, all request/response structs |
| T-P202 | `LinearClient` struct, `do()` HTTP transport with Bearer auth, GraphQL error detection on 200, `fetchAll()` cursor-based paginator |
| T-P203 | `normalizeIssue()`: labels lowercase, blockers from relations, nil-safe fields, timestamps parsed |
| T-P204 | All 6 operations: FetchCandidateIssues (dedup), FetchIssuesByStates, FetchIssueStatesByIDs, FetchIssue, AssignIssue, UnassignIssue + identity resolution |
| T-P205 | Integration smoke test (`//go:build integration`) with env var gating |

### Phase 3: Workspace Manager + Hooks (5 tickets)

Workspace lifecycle with path safety, hook execution, and cleanup.

| Ticket | What It Does |
| --- | --- |
| T-P300 | `Manager` with `ResolvePath` (sanitize + `filepath.Rel` containment), `EnsureDir` (new vs existing detection) |
| T-P301 | `RunHook`: bash -lc execution, process group isolation (`Setpgid`), timeout with `Kill(-pid)`, env var injection, output capture |
| T-P302 | `Prepare`: resolve → ensure → after_create (new only, removes dir on failure) → before_run (preserves dir on failure) |
| T-P303 | `CleanForIssue` and `CleanTerminal`: containment-verified removal, idempotent |
| T-P304 | `tempad clean <identifier>` wired to workspace manager, config-driven workspace root |

---

## Current State

```bash
cd code/go
go build ./cmd/tempad       # compiles
go test -race ./...          # all 57 tests pass
go vet ./...                 # clean

./tempad --help              # prints usage
./tempad init                # creates ~/.tempad/config.yaml
./tempad validate            # loads + merges + validates config
./tempad clean ABC-123       # removes workspace for ABC-123
```

### Stats

- **Total Go LOC:** ~5,649 (production + tests)
- **Production files:** 31
- **Test files:** 17
- **Tickets done:** 21/57 (37%)

---

## Source Files Map

### Phase 2 Files (Tracker)

```text
internal/tracker/
  client.go                  Client interface (6 methods)
  errors.go                  9 typed error types (including RateLimitError)
  errors_test.go             10 error tests

internal/tracker/linear/
  graphql.go                 8 queries/mutations, IssueFields fragment, all response types
  client.go                  LinearClient struct, Config, NewLinearClient, do() transport
  pagination.go              fetchAll() generic cursor-based paginator
  normalize.go               normalizeIssue(), normalizeIssues()
  operations.go              All 6 tracker.Client methods + ResolveIdentity
  client_test.go             7 transport tests (auth, errors, rate limit, pagination)
  normalize_test.go          12 normalization tests
  operations_test.go         9 operation tests
  integration_test.go        Integration smoke test (build-tagged)
```

### Phase 3 Files (Workspace)

```text
internal/workspace/
  manager.go                 Manager, ResolvePath, EnsureDir, HookConfig, Prepare,
                             CleanForIssue, CleanTerminal, verifyContainment
  hooks.go                   RunHook with process group kill and timeout
  manager_test.go            13 path resolution + safety tests
  hooks_test.go              9 hook execution tests
  prepare_test.go            8 lifecycle tests
  cleanup_test.go            6 cleanup tests

cmd/tempad/
  clean.go                   tempad clean CLI command (updated)
```

---

## Git State

```text
Branch: feat/p3-04-clean-cli (latest code)
PRs: #1 through #11 open, chained sequentially
Merge order: #1 → #2 → #3 → ... → #11 (each based on previous)
```

### PR Chain

| PR | Branch | Title |
| --- | --- | --- |
| #1 | feat/p2-00-tracker-interface | feat(tracker): verify client interface and add error type tests |
| #2 | feat/p2-01-graphql-queries | feat(linear): add GraphQL query and mutation builders |
| #3 | feat/p2-02-http-transport | feat(linear): add HTTP transport and cursor-based pagination |
| #4 | feat/p2-03-issue-normalization | feat(linear): add issue normalization |
| #5 | feat/p2-04-tracker-operations | feat(linear): implement all 6 tracker.Client operations |
| #6 | feat/p2-05-tracker-smoke-test | test(linear): add integration smoke test |
| #7 | feat/p3-00-workspace-paths | feat(workspace): add path resolution and safety |
| #8 | feat/p3-01-hook-execution | feat(workspace): add hook execution engine |
| #9 | feat/p3-02-workspace-prepare | feat(workspace): add Prepare lifecycle |
| #10 | feat/p3-03-workspace-cleanup | feat(workspace): add cleanup |
| #11 | feat/p3-04-clean-cli | feat(cli): wire up tempad clean |

---

## New Dependencies Added

None. Phase 2 and 3 use only stdlib (`net/http`, `encoding/json`, `os/exec`, `syscall`, `filepath`).

Phase 4 will need:
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — TUI styling
- `github.com/charmbracelet/bubbles` — TUI components (list, viewport)

---

## Workflow for Next Agent

### Task Execution Cycle

For each ticket T-PXYZ:

```bash
# 1. Branch from the PREVIOUS task's branch (not main)
git checkout feat/pX-YY-previous-task
git checkout -b feat/pX-YY-short-description

# 2. Move task to in-progress
git mv kanban/todo/pX-YY-name.md kanban/in-progress/

# 3. Read the task file — it has: problem, solution, files, work items,
#    acceptance criteria, and Research Notes at the bottom
cat kanban/in-progress/pX-YY-name.md

# 4. Read the Research Notes section at the bottom of the task
#    Read kanban/handoffs/research-findings.md for the relevant phase

# 5. Implement and test
#    go build ./cmd/tempad && go test -race ./... && go vet ./...

# 6. Move task to done
git mv kanban/in-progress/pX-YY-name.md kanban/done/

# 7. Commit (NO AI/Claude references in messages)
git add <files>
git commit -m "feat(scope): description

Implements T-PXYZ."

# 8. Push and create PR (base = previous task's branch)
git push -u origin feat/pX-YY-short-description
gh pr create --title "feat(scope): description" \
  --body "..." --base feat/pX-YY-previous-task
```

### PR Chain Rule

Every PR is based on the previous task's branch, forming a chain:

```text
main ← PR#1 ← PR#2 ← PR#3 ← ... ← PR#11 ← PR#12 ← PR#13 ← ...
```

The FIRST task of Phase 4 (T-P400) should branch from `feat/p3-04-clean-cli` (PR #11).

### End of Phase

At the end of each phase:
1. Write a handoff document: `kanban/handoffs/phase-X-handoff.md`
2. Include: what was built, source file map, git state, what's next
3. Continue to the next phase — same chain of PRs

### When Confused

1. Read the task file's **Research Notes** section
2. Read `kanban/handoffs/research-findings.md` for the relevant phase
3. Read `docs/SPEC_v1.md` sections cited in the task's `Spec:` field
4. Read `docs/ARCHITECTURE_GO_v1.md` for structural guidance
5. If a technical decision is unclear, research before implementing

### Rules (from AGENTS.md)

1. Spec is source of truth — implementation must not conflict
2. Keep changes narrowly scoped to the current task
3. Every task has acceptance criteria — all must pass before done
4. Run `go build ./cmd/tempad && go test -race ./...` after every change
5. No AI references in commits, PRs, or branch names
6. Update README when adding new packages or changing public APIs
7. Keep docs precise and minimal

---

## What Phase 4 Needs to Build

**Goal:** `tempad` (default, no flags) shows a live task board from Linear, lets developer select → claim → workspace → IDE.

### New Dependencies to Add

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

### Execution Order

```text
T-P400 → T-P401 → {T-P402, T-P404} → T-P403 → T-P405 → T-P406 → T-P407 → T-P408
```

| Ticket | Task | Key Details |
| --- | --- | --- |
| T-P400 | Claim mechanism | `internal/claim/claimer.go` — assign → fetch → verify → ClaimConflictError on race |
| T-P401 | Bubble Tea app model | `internal/tui/app.go` + `messages.go` — Model struct, message types, Init() |
| T-P402 | Board view rendering | Task list with priority sorting, [BLOCKED] markers, Available/Active sections |
| T-P403 | Keyboard navigation | j/k/Enter/r/d/o/u/q keybindings |
| T-P404 | Detail view | All issue fields, description, blockers (can parallelize with T-P402) |
| T-P405 | Poll loop | Periodic refresh with `pollInFlight` dedup flag |
| T-P406 | Selection flow | claim → workspace.Prepare → open IDE |
| T-P407 | Release task | u key unassigns via claim.Release |
| T-P408 | TUI entry point | Wire into `cmd/tempad/main.go` RunE |

### Research Findings for Phase 4

- **Poll dedup**: Add `pollInFlight bool` flag — only fire poll Cmd if `!m.pollInFlight`, set on poll start, clear on PollResultMsg
- **Model composition**: Use embedded sub-models per view + `viewState` enum, not flat struct
- **Selection preservation**: Store selected issue ID, re-find by ID after poll refresh
- Use `lipgloss` for styling, `bubbles/list` for task board
- Use `teatest` for headless TUI testing
- Handle `tea.WindowSizeMsg` for responsive layout

---

## What Phase 5 Can Build in Parallel

Phase 5 (Daemon Mode) is independent of Phase 4. Both can proceed after Phase 3.

Phase 5 target: `tempad --daemon` runs fully autonomous: poll → claim → dispatch → monitor → retry → reconcile. 14 tickets (T-P500 through T-P513).

Key research for Phase 5:
- Channel buffers: `make(chan WorkerResult, maxConcurrent)` to prevent goroutine leaks
- Retry timers: `time.AfterFunc` callbacks must check `ctx.Err()` before state mutation
- Subprocess: `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pid, sig)`
- Use `cenkalti/backoff/v4` for exponential backoff with jitter
- Use `signal.NotifyContext` for clean signal handling

---

## How to Start Phase 4

```bash
# 1. Verify everything is solid
cd code/go
go build ./cmd/tempad && go test -race ./... && go vet ./...

# 2. Read the Phase 4 plan
cat kanban/plans/phase-4-tui.md

# 3. Read the first task
cat kanban/todo/p4-00-claim-mechanism.md

# 4. Read research findings
cat kanban/handoffs/research-findings.md

# 5. Branch from the last PR and start
git checkout feat/p3-04-clean-cli
git checkout -b feat/p4-00-claim-mechanism
git mv kanban/todo/p4-00-claim-mechanism.md kanban/in-progress/

# 6. Add Bubble Tea dependencies (needed for T-P401)
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```
