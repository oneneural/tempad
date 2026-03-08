# Phase 1 Handoff: Foundation Complete

**Date:** 2026-03-08
**Status:** Phase 1 COMPLETE (10/10 tickets). Ready for Phase 2 development.

---

## What Was Built

Phase 1 delivers the foundation: all domain types, the 5-level config system, CLI scaffolding, prompt builder, and integration tests. The binary compiles and the three CLI commands work.

### Completed Tickets

| Ticket | What It Does |
| --- | --- |
| T-P100 | Go module scaffold (`go.mod`, directory structure, all stub packages) |
| T-P101 | Domain model: `Issue` (14 fields), `BlockerRef`, `Workspace`, `RunAttempt`, `RetryEntry`, `OrchestratorState` |
| T-P102 | Workflow loader: parses YAML front matter from `WORKFLOW.md`, extracts prompt body |
| T-P103 | User config loader: reads `~/.tempad/config.yaml` with YAML struct tags |
| T-P104 | Environment variable resolution: `$VAR` substitution, `~` expansion |
| T-P105 | `ServiceConfig` struct (33 fields) + 5-level merge (CLI > User > Repo > Env > Defaults) |
| T-P106 | Dispatch preflight validation: checks tracker kind, API key, project slug, identity, active states vs terminal states overlap |
| T-P107 | Cobra CLI: `tempad`, `tempad init`, `tempad validate`, `tempad clean` |
| T-P108 | Prompt builder: Liquid template rendering with `StrictVariables` mode |
| T-P109 | Integration test: full config pipeline (workflow + user config + env vars + merge + validate) |

### What Works Now

```bash
cd code/go
go build ./cmd/tempad       # compiles
go test -race ./...          # all tests pass
go vet ./...                 # clean

./tempad --help              # prints usage
./tempad init                # creates ~/.tempad/config.yaml
./tempad validate            # loads WORKFLOW.md, merges config, validates
./tempad clean               # placeholder (needs tracker client in Phase 2)
```

---

## Source Files Map

### Production Code

```text
cmd/tempad/
  main.go                    Root Cobra command, --daemon flag, mode selection
  init.go                    tempad init: scaffold ~/.tempad/config.yaml
  validate.go                tempad validate: load + merge + validate config
  clean.go                   tempad clean: stub (needs tracker.Client)

internal/domain/
  issue.go                   Issue struct (14 fields), BlockerRef, HasNonTerminalBlockers()
  workspace.go               Workspace struct (path, issue ID, created timestamp)
  run.go                     RunAttempt, RetryEntry structs
  state.go                   OrchestratorState struct (running map, retry queue)
  normalize.go               NormalizeState(), SanitizeIdentifier() for path safety

internal/config/
  config.go                  ServiceConfig (33 fields), CLIFlags, Defaults()
  workflow.go                LoadWorkflow(): YAML front matter + prompt body parser
  user.go                    LoadUserConfig(): ~/.tempad/config.yaml loader
  resolve.go                 ResolveEnvVars(): $VAR substitution, ExpandHome(): ~ expansion
  loader.go                  LoadAndMerge(): full 5-level merge pipeline
  validation.go              ValidateForDispatch(): preflight checks

internal/prompt/
  builder.go                 RenderPrompt(): Liquid template with StrictVariables
```

### Test Code

```text
internal/domain/normalize_test.go      State normalization, identifier sanitization
internal/config/workflow_test.go       YAML front matter parsing edge cases
internal/config/user_test.go           User config loading
internal/config/resolve_test.go        Env var substitution, ~ expansion
internal/config/loader_test.go         5-level merge precedence
internal/config/validation_test.go     Dispatch preflight validation
internal/config/integration_test.go    Full pipeline integration test
internal/prompt/builder_test.go        Liquid rendering, strict vars, filters
```

### Stub Files (Scaffolded for Future Phases)

These exist with package declarations only — no implementation:

```text
internal/tracker/client.go             tracker.Client interface (6 operations) — READY
internal/tracker/errors.go             7 typed error types — READY
internal/tracker/linear/client.go      Empty — Phase 2 target
internal/workspace/manager.go          Empty — Phase 3 target
internal/agent/launcher.go             Empty — Phase 5 target
internal/claim/claimer.go              Empty — Phase 4 target
internal/orchestrator/orchestrator.go   Empty — Phase 5 target
internal/tui/app.go                    Empty — Phase 4 target
internal/server/server.go              Empty — Phase 7 target
internal/logging/setup.go              Empty — Phase 6 target
```

### Stats

- **Total Go LOC:** ~2,879 (including tests)
- **Test files:** 8
- **Production files:** 18 (10 implemented + 8 stubs)

---

## Dependencies (go.mod)

```text
github.com/osteele/liquid v1.4.0       Liquid template engine (StrictVariables)
github.com/spf13/cobra v1.8.1          CLI framework
github.com/stretchr/testify v1.9.0     Test assertions (require, assert)
gopkg.in/yaml.v3 v3.0.1                YAML parsing
```

---

## Key Types the Next Phase Needs

### `domain.Issue` (internal/domain/issue.go)

14-field struct. The tracker client must normalize Linear API responses into this shape.

Key fields: `ID` (Linear UUID), `Identifier` ("ABC-123"), `Title`, `Description`, `Priority` (*int, lower = higher), `State` (string), `Assignee` (email or user ID, "" = unassigned), `Labels` ([]string, lowercase), `BlockedBy` ([]BlockerRef), `URL`, `CreatedAt`, `UpdatedAt`.

### `tracker.Client` Interface (internal/tracker/client.go)

Already defined with 6 operations:

1. `FetchCandidateIssues(ctx) → ([]Issue, error)` — unassigned issues in active states
2. `FetchIssueStatesByIDs(ctx, ids) → (map[string]string, error)` — for reconciliation
3. `FetchIssuesByStates(ctx, states) → ([]Issue, error)` — for startup cleanup
4. `FetchIssue(ctx, id) → (*Issue, error)` — for claim verification
5. `AssignIssue(ctx, issueID, identity) → error` — claim
6. `UnassignIssue(ctx, issueID) → error` — release claim

### Typed Errors (internal/tracker/errors.go)

Already defined: `UnsupportedTrackerKindError`, `MissingTrackerAPIKeyError`, `MissingTrackerProjectSlugError`, `MissingTrackerIdentityError`, `APIRequestError` (wraps transport), `APIStatusError` (non-200), `APIErrorsError` (GraphQL errors array), `ClaimConflictError`.

### `config.ServiceConfig` (internal/config/config.go)

Tracker-relevant fields the Linear client needs:

- `TrackerEndpoint` — default `https://api.linear.app/graphql`
- `TrackerAPIKey` — Linear API key
- `TrackerProjectSlug` — project slug (NOT slugId)
- `TrackerIdentity` — email for identity resolution
- `ActiveStates` — default `["Todo", "In Progress"]`
- `TerminalStates` — default `["Closed", "Cancelled", "Canceled", "Duplicate", "Done"]`

---

## What Phase 2 Needs to Build

**Goal:** Implement `tracker.Client` for Linear's GraphQL API. All 6 operations working.

### Files to Create/Modify

```text
internal/tracker/linear/
  client.go          LinearClient struct, constructor, config
  graphql.go         Query/mutation string builders
  normalize.go       Linear API response → domain.Issue normalization
  pagination.go      Cursor-based pagination helper
  client_test.go     Unit tests with httptest mock server
```

### Phase 2 Tickets (in order)

```text
{T-P200, T-P201} → T-P202 → T-P203 → T-P204 → T-P205
```

1. **T-P200** — Tracker client interface + error types (ALREADY DONE in Phase 1 stubs — verify and enhance)
2. **T-P201** — GraphQL query/mutation string builders
3. **T-P202** — HTTP transport + cursor-based pagination
4. **T-P203** — Issue normalization (Linear → domain.Issue)
5. **T-P204** — Implement all 6 tracker operations
6. **T-P205** — Integration smoke test against real Linear API

### Critical Research Notes

Read `kanban/handoffs/research-findings.md` for validated patterns. Key items:

1. **Use `project.slug` NOT `slugId`** — `slugId` is deprecated in Linear's API
2. **Linear returns HTTP 200 for GraphQL errors** — must parse `errors[]` from response body even on 200
3. **Rate limit:** 5,000 requests/hour per API key (complexity-weighted)
4. **Identity resolution:** Use `users(filter: { email: { eq: "..." } })` to resolve email → Linear user ID. Cache at client construction.
5. **Auth:** `Authorization: Bearer <api_key>` header
6. **Pagination:** Cursor-based with `first`/`after` and `pageInfo { hasNextPage, endCursor }`

### Constructor Pattern

```go
// LinearClient should be constructed with config fields:
type LinearClient struct {
    endpoint    string          // from ServiceConfig.TrackerEndpoint
    apiKey      string          // from ServiceConfig.TrackerAPIKey
    projectSlug string          // from ServiceConfig.TrackerProjectSlug
    identity    string          // from ServiceConfig.TrackerIdentity (email)
    userID      string          // resolved from identity via API (cached)
    httpClient  *http.Client
    activeStates   []string
    terminalStates []string
}
```

---

## What Phase 3 Can Build in Parallel

Phase 3 (Workspace Manager) has no dependency on Phase 2. Both can proceed simultaneously.

Phase 3 target files: `internal/workspace/manager.go`, `hooks.go`, `cleanup.go`.

---

## Documentation Changes Made in This Session

Beyond Phase 1 code, we also completed documentation cleanup:

1. **Root README.md** — restructured with separate "Specification & Design" and "Implementations" sections, added Progress column, "Adding a New Implementation" guide
2. **code/go/README.md** — moved dev warning callout here, fixed `init.go` description
3. **Markdown lint** — added `.markdownlint.json` config, fixed all 1,711 lint errors to 0 across 55 files (MD022, MD031, MD032, MD040, MD060 rules)
4. **4 commits** on `main` (ahead of origin by 4, behind by 1 — needs `git pull --rebase` before push)

---

## How to Start Phase 2

```bash
# 1. Verify Phase 1 is solid
cd code/go
go build ./cmd/tempad && go test -race ./... && go vet ./...

# 2. Read the Phase 2 plan
cat kanban/plans/phase-2-tracker.md

# 3. Read the first task
cat kanban/todo/p2-00-tracker-interface.md

# 4. Read research findings for Phase 2 section
cat kanban/handoffs/research-findings.md

# 5. Move first task to in-progress and start coding
git mv kanban/todo/p2-00-tracker-interface.md kanban/in-progress/
```

---

## Git State

```text
Branch: main
Ahead of origin: 4 commits (docs cleanup)
Behind origin: 1 commit (needs git pull --rebase)
Working tree: clean
```
