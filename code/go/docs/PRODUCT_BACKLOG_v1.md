# T.E.M.P.A.D. Product Backlog

**Temporal Execution & Management Poll-Agent Dispatcher**

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Module** | `github.com/oneneural/tempad` |
| **Date** | 2026-03-08 |
| **Derived from** | `SPEC_v1.md` (v1.0.0), `ARCHITECTURE_GO_v1.md` (v1.0.0), `STACK_COMPARISON_v1.md` (v1.0.0) |

---

## 1. Project Overview

T.E.M.P.A.D. is a developer-local service that continuously reads work from an issue tracker (Linear), presents available tasks to the developer, and either opens an IDE session for the selected task or runs a coding agent headlessly in an isolated workspace. It is an enhanced open-source alternative to OpenAI's Symphony.

There is no central server. Each developer runs T.E.M.P.A.D. on their own machine. The issue tracker is the shared coordination layer — task assignment prevents duplicate work across the team.

### What It Solves

- Live view of available work so developers can choose what to pick up
- Repeatable workflow execution instead of manual scripts
- Per-issue workspace isolation so agent commands run only inside their workspace
- In-repo workflow policy (`WORKFLOW.md`) so teams version agent prompts and runtime settings with their code
- Enough observability to operate and debug concurrent agent runs

### What It Doesn't Do

- No central server, fleet management, or multi-node coordination
- No rich web UI or multi-tenant control plane
- No built-in business logic for editing tickets, PRs, or comments (that lives in the workflow prompt and agent tooling)
- No mandated coding agent or IDE — agent-agnostic by design

### Operating Modes

**TUI Mode (Interactive, default):** Developer sees a live task board, selects a task, T.E.M.P.A.D. claims it via Linear assignment, creates/reuses a per-issue workspace, runs hooks, opens the IDE. Developer and their agent work inside the IDE from there.

**Daemon Mode (Headless, `--daemon`):** T.E.M.P.A.D. auto-selects tasks, claims them, creates workspaces, launches configured coding agents as subprocesses, manages the full lifecycle (monitoring, retry with exponential backoff, stall timeouts, concurrency limits, reconciliation).

---

## 2. Tech Stack Decision

**Go 1.22+** was chosen after a weighted comparison against Rust and Elixir.

### Final Scores (Weighted)

| Stack | Raw Score | Weighted Score |
| ------- | ----------- | --------------- |
| **Go** | 58/60 | **116** |
| Rust | 50/60 | 101 |
| Elixir | 48/60 | 93 |

### Key Reasons for Go

- **Bubble Tea** is the best TUI framework across any language — purpose-built, mature, beautiful defaults
- **Distribution** is trivial: single static binary via `GOOS=darwin GOARCH=arm64 go build`
- **Goroutines + channels** are the exact concurrency model the orchestrator needs (poll loop, N agent workers, retry timers via `time.AfterFunc`)
- **osteele/liquid** has strict variable mode out of the box — Rust's liquid crate lacks this spec requirement
- **slog** in stdlib means zero-dep structured logging
- Fast compile-test cycle accelerates development

### Dependencies

| Purpose | Package | Notes |
| --------- | --------- | ------- |
| CLI framework | `github.com/spf13/cobra` | Industry standard |
| TUI framework | `github.com/charmbracelet/bubbletea` | Elm Architecture, v1.x |
| TUI styling | `github.com/charmbracelet/lipgloss` | Pairs with Bubble Tea |
| TUI components | `github.com/charmbracelet/bubbles` | List, spinner, textinput |
| Liquid templates | `github.com/osteele/liquid` | Strict vars mode |
| YAML parsing | `gopkg.in/yaml.v3` | Mature, pure Go |
| File watching | `github.com/fsnotify/fsnotify` | Cross-platform, v1.x |
| HTTP router | `github.com/go-chi/chi/v5` | Minimal, middleware-friendly |
| Structured logging | `log/slog` (stdlib) | Go 1.22+ built-in |
| Goroutine leak test | `go.uber.org/goleak` | Test only |
| Test assertions | `github.com/stretchr/testify` | Test only |

Stdlib covers: HTTP client (`net/http`), JSON (`encoding/json`), subprocess (`os/exec`), context/cancellation (`context`), timers (`time`), filepath (`path/filepath`).

---

## 3. Architecture Summary

### Design Principles

1. **Shared core, thin mode adapters.** Tracker client, workspace manager, config layer, claim logic, and prompt builder are shared. TUI and daemon are adapters that compose these differently.
2. **Channels for coordination.** Agent workers send results on a shared channel. The orchestrator's `select` loop is the single point of state mutation. No shared mutable state between goroutines.
3. **Interfaces at boundaries.** `tracker.Client` is an interface so Linear is swappable. Agent launcher is an interface so test doubles work. Everything else is concrete.
4. **Fail loud at startup, recover gracefully at runtime.** Missing config fails startup with a clear error. Tracker API errors during operation skip the current tick and retry next.
5. **The spec is the source of truth.** The architecture doc describes how to build what the spec says.

### Directory Structure

```text
code/go/
├── cmd/tempad/
│   ├── main.go            # Entry point: parse flags, select mode, run
│   ├── init.go            # tempad init command
│   ├── validate.go        # tempad validate command
│   └── clean.go           # tempad clean command
├── internal/
│   ├── config/            # Config layer (load, merge, validate, watch)
│   │   ├── config.go      # ServiceConfig struct, merge logic, defaults
│   │   ├── user.go        # UserConfig (~/.tempad/config.yaml) loader
│   │   ├── workflow.go    # WorkflowDefinition (YAML front matter + prompt)
│   │   ├── loader.go      # Load + merge + validate pipeline
│   │   ├── resolve.go     # $VAR resolution, ~ expansion
│   │   ├── validation.go  # Dispatch preflight checks
│   │   └── watcher.go     # fsnotify WORKFLOW.md hot reload
│   ├── domain/            # Core domain model (no infra deps)
│   │   ├── issue.go       # Issue struct (14 fields)
│   │   ├── workspace.go   # Workspace struct
│   │   ├── run.go         # RunAttempt, RetryEntry structs
│   │   ├── state.go       # OrchestratorState struct
│   │   └── normalize.go   # SanitizeIdentifier, NormalizeState
│   ├── tracker/           # Issue tracker abstraction
│   │   ├── client.go      # Client interface (6 operations)
│   │   ├── errors.go      # Typed error categories
│   │   └── linear/        # Linear implementation
│   │       ├── client.go
│   │       ├── graphql.go
│   │       ├── normalize.go
│   │       └── pagination.go
│   ├── workspace/         # Workspace lifecycle
│   │   ├── manager.go     # Create, reuse, path resolution, safety
│   │   ├── hooks.go       # Hook execution (bash -lc, timeout)
│   │   └── cleanup.go     # Terminal workspace cleanup
│   ├── prompt/            # Prompt construction
│   │   └── builder.go     # Liquid template rendering (strict vars)
│   ├── agent/             # Agent launcher
│   │   ├── launcher.go    # Launcher interface + subprocess impl
│   │   ├── process.go     # Process management: spawn, wait, kill
│   │   ├── delivery.go    # Prompt delivery: file, stdin, arg, env
│   │   └── output.go      # Stdout/stderr capture, JSON parsing
│   ├── claim/             # Claim mechanism (shared TUI + daemon)
│   │   └── claimer.go     # Assign → verify → release on conflict
│   ├── orchestrator/      # Daemon mode orchestrator
│   │   ├── orchestrator.go # Main run loop, tick, select
│   │   ├── dispatch.go    # Candidate selection, sorting, dispatch
│   │   ├── reconcile.go   # Stall detection + tracker state refresh
│   │   ├── retry.go       # Retry scheduling, backoff
│   │   └── worker.go      # Agent worker goroutine
│   ├── tui/               # TUI mode (Bubble Tea)
│   │   ├── app.go         # tea.Model: root application model
│   │   ├── board.go       # Task board view
│   │   ├── detail.go      # Task detail view
│   │   ├── keys.go        # Key bindings
│   │   ├── styles.go      # Lip Gloss styles
│   │   └── messages.go    # Custom tea.Msg types
│   ├── server/            # Optional HTTP server
│   │   ├── server.go      # Chi router, loopback bind
│   │   └── handlers.go    # API endpoints
│   └── logging/           # Logging setup
│       ├── setup.go       # slog handler configuration
│       └── rotate.go      # Log rotation (size-based)
├── go.mod
├── go.sum
└── README.md
```

### Package Dependency Rule

No circular dependencies. `domain/` is a leaf node with zero imports. All arrows flow downward toward `domain/`.

### Core Interfaces

**Tracker Client** — 6 operations: `FetchCandidateIssues`, `FetchIssueStatesByIDs`, `FetchIssuesByStates`, `FetchIssue`, `AssignIssue`, `UnassignIssue`

**Agent Launcher** — `Launch(ctx, opts) → RunHandle` with `Wait()`, `Cancel()`, `Stdout`, `Stderr`

**Workspace Manager** — `Prepare(ctx, issue, hooks) → Workspace`, `CleanForIssue(ctx, identifier)`, `CleanTerminal(ctx, terminalIssues)`

### Configuration Architecture

Five-level merge precedence (highest wins): CLI flags → User config (`~/.tempad/config.yaml`) → Repo config (`WORKFLOW.md` front matter) → Environment variable indirection (`$VAR`) → Built-in defaults

Personal fields (identity, api_key, IDE, agent command) → user config wins.
Team fields (hooks, states, workspace root, concurrency) → repo config wins.

ServiceConfig has 33 typed fields covering tracker, polling, workspace, hooks, agent lifecycle, IDE, display, logging, and HTTP server settings.

### Concurrency Model

**Daemon Mode:** Orchestrator owns all mutable state via a `select` loop over: `ctx.Done()` (shutdown), `ticker.C` (poll tick), `workerResults` (agent exits), `retryTimers` (retry fires), `configReload` (hot reload). Workers communicate only via channels.

**TUI Mode:** Bubble Tea event loop receives messages (keyboard, poll results, claim outcomes) and returns updated state + commands. All state lives in `tea.Model`. Background work runs via `tea.Cmd` functions.

---

## 4. Setup Guide

### Prerequisites

- Go 1.22+ installed
- Linear account with API key
- A coding agent (for daemon mode): Claude Code, Codex, OpenCode, etc.
- An IDE (for TUI mode): VS Code, Cursor, Zed, etc.

### Build from Source

```bash
cd code/go
go mod tidy
go build -o tempad ./cmd/tempad
```

### First Run

```bash
# Initialize personal config
./tempad init
# This creates ~/.tempad/config.yaml with commented defaults

# Edit your config
vim ~/.tempad/config.yaml
# Set tracker.identity, tracker.api_key ($LINEAR_API_KEY), ide.command, agent.command

# Create WORKFLOW.md in your repo root (see Spec Appendix A for example)

# Validate configuration
./tempad validate

# Run TUI mode (interactive)
./tempad

# Run daemon mode (headless)
./tempad --daemon --workflow ./WORKFLOW.md
```

### CLI Flags

| Flag | Description | Default |
| ------ | ------------- | --------- |
| `--daemon` | Run in headless daemon mode | false (TUI mode) |
| `--workflow <path>` | Path to WORKFLOW.md | `./WORKFLOW.md` |
| `--identity <id>` | Tracker identity (overrides config) | from config |
| `--agent <cmd>` | Agent command (overrides config) | from config |
| `--ide <cmd>` | IDE command (overrides config) | `code` |
| `--port <port>` | HTTP server port (daemon mode only) | 0 (disabled) |
| `--log-level <level>` | Log level (debug/info/warn/error) | `info` |

### Running Tests

```bash
# Unit tests
go test ./...

# With race detection
go test -race ./...

# Integration tests (requires LINEAR_API_KEY)
go test -tags=integration ./...

# Smoke tests (requires real Linear project)
go test -tags=smoke ./test/...
```

### Cross-Compilation

```bash
# macOS ARM
GOOS=darwin GOARCH=arm64 go build -o tempad-darwin-arm64 ./cmd/tempad

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o tempad-darwin-amd64 ./cmd/tempad

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o tempad-linux-amd64 ./cmd/tempad
```

---

## 5. Phase Overview and Status

| Phase | Tickets | Status | Key Deliverable |
| ------- | --------- | -------- | ---------------- |
| 1 — Foundation | T-P100 to T-P109 (10) | **COMPLETE** | CLI, config pipeline, domain model, prompt builder |
| 2 — Tracker Client | T-P200 to T-P205 (6) | Pending | Linear GraphQL client, all 6 operations, normalization |
| 3 — Workspace Manager | T-P300 to T-P304 (5) | Pending | Workspace lifecycle, hooks, safety, `tempad clean` |
| 4 — TUI Mode | T-P400 to T-P408 (9) | Pending | Interactive task board, claim flow, IDE launch |
| 5 — Daemon Mode | T-P500 to T-P513 (14) | Pending | Full orchestrator: dispatch, retry, reconcile, agent lifecycle |
| 6 — Hot Reload + Logging | T-P600 to T-P603 (4) | Pending | Dynamic config reload, structured logging |
| 7 — HTTP Server | T-P700 to T-P702 (3) | Pending | REST API and dashboard for daemon observability |
| 8 — Testing + Hardening | T-P800 to T-P805 (6) | Pending | Race detection, goroutine leaks, e2e, smoke tests |
| **Total** | **57 tickets** | | |

### Phase Dependency Graph

```text
Phase 1 (Foundation) ──┬──→ Phase 2 (Tracker) ──┬──→ Phase 4 (TUI) ──────┬──→ Phase 6 (Polish) ──┬──→ Phase 8 (Hardening)
                       │                         │                         │                        │
                       └──→ Phase 3 (Workspace) ─┤──→ Phase 5 (Daemon) ──┼──→ Phase 7 (HTTP) ─────┘
                                                  │                        │
                                                  └────────────────────────┘
```

Key parallelization: Phases 2 and 3 can be developed in parallel after Phase 1. Phase 4 (TUI) and Phase 5 (Daemon) each need both 2 and 3 but are independent of each other. Phase 7 (HTTP Server) only depends on Phase 5.

---

## 6. Phase 1: Foundation — COMPLETE

**Goal:** `tempad init` and `tempad validate` work. All domain types defined. Config loads, merges, validates.

All 10 tickets have been implemented with source code and unit tests.

### T-P100: Initialize Go module and project scaffold ✅

- **Files:** `go.mod`, `cmd/tempad/main.go`, full `internal/` directory tree, `.gitignore`
- **Spec:** Section 18.1
- **What was built:**
  - Go module `github.com/oneneural/tempad` with Cobra, Liquid, testify, yaml.v3 dependencies
  - Full directory structure matching architecture doc Section 3
  - Cobra root command with all flags (`--daemon`, `--workflow`, `--identity`, `--agent`, `--ide`, `--port`, `--log-level`)
  - `.gitignore` for Go binaries, IDE files, OS files
  - Stub files for all future phase packages

### T-P101: Define all domain model structs ✅

- **Files:** `internal/domain/issue.go`, `workspace.go`, `run.go`, `state.go`, `normalize.go`, `normalize_test.go`
- **Spec:** Section 4.1, 4.2
- **What was built:**
  - `Issue` struct with all 14 fields (id, identifier, title, description, priority, state, assignee, branch_name, url, labels, blocked_by, created_at, updated_at)
  - `BlockerRef` struct (id, identifier, state) with `HasNonTerminalBlockers` method
  - `Workspace` struct (Path, WorkspaceKey, CreatedNow)
  - `RunAttempt` struct (IssueID, IssueIdentifier, Attempt, WorkspacePath, StartedAt, Status, Error)
  - `RetryEntry` struct (IssueID, Identifier, Attempt, DueAtMs, TimerHandle, Error)
  - `OrchestratorState` with `NewOrchestratorState()`, `Snapshot()`, `RunningCount()`, `IsClaimedOrRunning()`, `AvailableSlots()`
  - `AgentTotals` struct (TotalInputTokens, TotalOutputTokens, TotalRuntimeSeconds)
  - `SanitizeIdentifier()` — replaces `[^A-Za-z0-9._-]` with `_`
  - `NormalizeState()` — trim + lowercase
  - `NormalizeStates()` — returns map for efficient lookup
  - Unit tests for sanitization and normalization

### T-P102: Workflow loader ✅

- **Files:** `internal/config/workflow.go`, `workflow_test.go`
- **Spec:** Section 6.1, 6.2, 6.3, 6.5
- **What was built:**
  - `LoadWorkflow(path)` with file discovery (explicit path or `./WORKFLOW.md` default)
  - YAML front matter parsing (`---` delimited)
  - No front matter → empty config, full file as prompt
  - Non-map YAML → `workflow_front_matter_not_a_map` error
  - Unknown keys ignored (forward compat)
  - `WorkflowError` type with `Code` field for structured error handling
  - Helper functions: `getNestedString`, `getNestedInt`, `getNestedStringList`, `getNestedIntMap`
  - 6 test cases (full, missing file, no front matter, non-map, empty, unknown keys)

### T-P103: User config loader ✅

- **Files:** `internal/config/user.go`, `user_test.go`
- **Spec:** Section 7.1, 7.2, 7.3
- **What was built:**
  - `LoadUserConfig(path)` with default path `~/.tempad/config.yaml`
  - `UserConfig` struct with nested structs (UserTrackerConfig, UserIDEConfig, UserAgentConfig, UserDisplayConfig, UserLoggingConfig)
  - Missing file → empty config without error
  - Malformed YAML → parse error
  - `$VAR` values preserved as-is (resolution happens in merge step)
  - `DefaultUserConfigPath()` and `DefaultUserConfigTemplate` constant
  - 4 unit tests

### T-P104: Environment variable resolution ✅

- **Files:** `internal/config/resolve.go`, `resolve_test.go`
- **Spec:** Section 8.1 (item 4), Section 6.3.1
- **What was built:**
  - `ResolveEnvVar(value)` — if starts with `$`, look up `os.Getenv`
  - Empty resolution treated as missing (returns `""`)
  - `ExpandHome(path)` — `~/path` expansion using `os.UserHomeDir()`
  - Unit tests with controlled env vars

### T-P105: ServiceConfig struct and merge logic ✅

- **Files:** `internal/config/config.go`, `loader.go`, `loader_test.go`
- **Spec:** Section 8.1, 8.4, 4.1.4
- **What was built:**
  - `ServiceConfig` struct with all 33 fields from Architecture Section 7.2
  - `CLIFlags` struct for command-line overrides
  - `Defaults()` function returning all built-in defaults per Spec 8.4
  - `Merge(cli, user, workflow)` with correct precedence: CLI > User > Repo > EnvVar > Defaults
  - Personal fields (identity, api_key, IDE, agent command) → user wins
  - Team fields (hooks, states, workspace root, concurrency) → repo wins
  - `$VAR` resolution applied after merge
  - `active_states`/`terminal_states` handled as list OR comma-separated string
  - `Load(workflowPath, userConfigPath, cliFlags)` pipeline function
  - 10 unit tests for merge precedence edge cases

### T-P106: Dispatch preflight validation ✅

- **Files:** `internal/config/validation.go`, `validation_test.go`
- **Spec:** Section 8.3
- **What was built:**
  - `ValidateForStartup(cfg, mode)` and `ValidateForDispatch(cfg, mode)`
  - `ValidationError` and `ValidationErrors` types
  - 6 validation checks: tracker.kind present and supported, api_key present after $VAR resolution, project_slug present (Linear), identity present, agent.command present (daemon only), workflow loadable
  - Human-readable error messages naming the specific field
  - 8 unit tests covering all checks

### T-P107: Cobra CLI with init and validate commands ✅

- **Files:** `cmd/tempad/main.go`, `init.go`, `validate.go`, `clean.go`
- **Spec:** Section 18.1, 18.2
- **What was built:**
  - Root command `tempad` with all flags
  - `tempad init` — creates `~/.tempad/config.yaml` with commented defaults, no overwrite
  - `tempad validate` — loads workflow + user config, merges, validates, prints result, exit 0/1
  - `tempad clean` — placeholder for Phase 3 workspace cleanup

### T-P108: Prompt builder with Liquid templates ✅

- **Files:** `internal/prompt/builder.go`, `builder_test.go`
- **Spec:** Section 6.4, Section 14
- **What was built:**
  - `Builder` struct with Liquid engine, `Render(template, issue, attempt)` method
  - `issueToMap()` converting `domain.Issue` to `map[string]any` for Liquid
  - All 14 fields accessible as `issue.id`, `issue.identifier`, etc.
  - `issue.labels` as list for `{% for %}` loops
  - `issue.blocked_by` as list of maps with id, identifier, state
  - `attempt` as int or nil
  - `DefaultPrompt` constant: `"Work on issue {{ issue.identifier }}: {{ issue.title }}"`
  - `default` filter verified working
  - 11 unit tests (basic, all fields, labels iteration, blocked_by, attempt nil/set, default filter, empty template, timestamps)

### T-P109: Phase 1 integration test ✅

- **Files:** `internal/config/integration_test.go`
- **Spec:** Section 20.1
- **What was built:**
  - Full pipeline test: temp WORKFLOW.md + temp config.yaml → load → merge → validate
  - Verifies correct merge, defaults, env var resolution
  - Exercises all Phase 1 components together

---

## 7. Phase 2: Tracker Client (Linear) — PENDING

**Goal:** All 6 tracker operations work against Linear's GraphQL API. Issues normalized into domain model.

**Prerequisite:** Phase 1 (complete)

### T-P200: Tracker client interface and error types

- **Files:** `internal/tracker/client.go`, `internal/tracker/errors.go`
- **Spec:** Section 13.1, 13.4
- **Status:** Interface and error types already stubbed in Phase 1
- **Work:**
  - Define `Client` interface with 6 methods:
    - `FetchCandidateIssues(ctx) ([]domain.Issue, error)`
    - `FetchIssueStatesByIDs(ctx, ids) (map[string]string, error)`
    - `FetchIssuesByStates(ctx, states) ([]domain.Issue, error)`
    - `FetchIssue(ctx, id) (*domain.Issue, error)`
    - `AssignIssue(ctx, issueID, identity) error`
    - `UnassignIssue(ctx, issueID) error`
  - Define typed errors: `UnsupportedTrackerKindError`, `MissingTrackerAPIKeyError`, `MissingTrackerProjectSlugError`, `MissingTrackerIdentityError`, `TrackerAPIRequestError` (transport), `TrackerAPIStatusError` (non-200), `TrackerAPIErrorsError` (GraphQL errors), `TrackerClaimConflictError` (assignment race)
  - All errors implement `error` and support `errors.Is` / `errors.As`
- **Acceptance:**
  - Interface compiles
  - Errors implement `error` with structured messages including context (issue ID, HTTP status)
- **Deps:** T-P101
- **Parallelizable:** Yes (with T-P201)

### T-P201: Linear GraphQL query/mutation builders

- **Files:** `internal/tracker/linear/graphql.go`
- **Spec:** Section 13.2, Architecture Section 8.1
- **Work:**
  - GraphQL query strings as Go constants:
    - `candidateIssuesQuery` — filter by project slug, active states, unassigned. Pagination. All Issue fields (id, identifier, title, description, priority, state.name, assignee.id/email, branchName, url, labels.nodes.name, relations for blockers, createdAt, updatedAt)
    - `assignedToMeQuery` — same but filter by current user (resumption)
    - `issueStatesByIDsQuery` — batch node lookup, return id + state.name
    - `issuesByStatesQuery` — filter by state names (terminal cleanup)
    - `singleIssueQuery` — one issue by ID (claim verification)
    - `assignIssueMutation` — `issueUpdate(id, input: { assigneeId })`
    - `unassignIssueMutation` — `issueUpdate(id, input: { assigneeId: null })`
  - Request/response structs for JSON marshaling
  - GraphQL error response parsing
- **Acceptance:**
  - All query strings are valid GraphQL (syntax check)
  - Response structs unmarshal real Linear responses correctly
  - GraphQL error format `{ errors: [{ message }] }` detected
- **Deps:** T-P100
- **Parallelizable:** Yes (with T-P200)

### T-P202: Linear HTTP transport and pagination

- **Files:** `internal/tracker/linear/client.go`, `internal/tracker/linear/pagination.go`
- **Spec:** Section 13.2
- **Work:**
  - `LinearClient` struct: httpClient, endpoint, apiKey, projectSlug, identity, timeout (30s)
  - `NewLinearClient(cfg)` constructor
  - `do(ctx, query, vars, result)` — POST to endpoint, `Authorization: Bearer <key>`, JSON body, unmarshal, check errors
  - Cursor-based pagination helper: `fetchAll[T](ctx, query, vars, extractPage)` loop until `hasNextPage` false, page size 50
  - HTTP timeout 30s, context cancellation respected
- **Acceptance:**
  - Correct `Authorization` header
  - Paginates correctly (mock server with 3 pages)
  - Context cancellation aborts in-flight request
  - Non-200 → `TrackerAPIStatusError`, network error → `TrackerAPIRequestError`
  - Unit tests with httptest mock server
- **Deps:** T-P200, T-P201

### T-P203: Issue normalization (Linear → domain.Issue)

- **Files:** `internal/tracker/linear/normalize.go`
- **Spec:** Section 13.3, 4.1.1
- **Work:**
  - `normalizeIssue(raw)` field mapping:
    - `labels` → lowercase all names
    - `blocked_by` → derive from inverse relations where type is `blocks`
    - `priority` → int only, non-int → nil
    - `assignee` → user ID or email
    - `created_at`, `updated_at` → parse ISO-8601
    - `branch_name` → from `branchName`
  - Handle nil/missing fields gracefully (no panics)
- **Acceptance:**
  - `["Bug", "Frontend"]` → `["bug", "frontend"]`
  - Priority `2` → `2`, priority `"high"` → nil
  - Relation `{type: "blocks", relatedIssue: {…}}` → `blocked_by` entry
  - Unit tests with real Linear response fixture
- **Deps:** T-P201, T-P101

### T-P204: Implement all 6 tracker operations

- **Files:** `internal/tracker/linear/client.go` (expand)
- **Spec:** Section 13.1
- **Work:**
  1. `FetchCandidateIssues(ctx)` — unassigned + active states + my assigned (resumption). Merge and deduplicate.
  2. `FetchIssuesByStates(ctx, states)` — terminal cleanup query
  3. `FetchIssueStatesByIDs(ctx, ids)` — batch node lookup → `map[id]state`
  4. `FetchIssue(ctx, id)` — single issue fetch
  5. `AssignIssue(ctx, issueID, identity)` — mutation
  6. `UnassignIssue(ctx, issueID)` — mutation with `assigneeId: null`
  - Identity resolution: if `identity` looks like email, resolve to Linear user ID at construction (cache). Use `viewer` query or `users(filter: {email})`.
- **Acceptance:**
  - Each operation handles success and error cases
  - `FetchCandidateIssues` returns normalized, deduplicated issues
  - Identity resolution from email works
  - 6 unit tests (one per operation) with mock server
- **Deps:** T-P202, T-P203

### T-P205: Tracker client integration smoke test

- **Files:** `internal/tracker/linear/integration_test.go`
- **Spec:** Section 20.3
- **Work:**
  - Test tagged `//go:build integration`
  - Requires `LINEAR_API_KEY` and `LINEAR_TEST_PROJECT_SLUG` env vars
  - Fetch candidates from real Linear project, verify normalization
  - Assign/unassign a designated test issue
  - Clean up after test
- **Acceptance:**
  - Passes against real Linear API
  - No orphaned assignments
  - Skips gracefully when env vars absent
- **Deps:** T-P204

### Phase 2 Internal Dependency Chain

```text
{T-P200, T-P201} → T-P202 → T-P203 → T-P204 → T-P205
```

---

## 8. Phase 3: Workspace Manager + Hooks — PENDING

**Goal:** Deterministic workspace creation, hook execution, safety invariants, cleanup.

**Prerequisite:** Phase 1 (complete). Phase 2 needed only for T-P304 (clean with tracker).

### T-P300: Workspace path resolution and safety invariants

- **Files:** `internal/workspace/manager.go`
- **Spec:** Section 12.1, 12.2, 12.6
- **Work:**
  - `NewManager(workspaceRoot)` constructor
  - `resolvePath(identifier)`: sanitize via `domain.SanitizeIdentifier()`, `filepath.Join(root, key)`, verify workspace path has root as prefix (reject path traversal `..`)
  - `ensureDir(path)`: `os.MkdirAll(path, 0755)`, detect newly created vs existing, error if non-directory file at path
- **Acceptance:**
  - `ABC-123` → `<root>/ABC-123`
  - `ABC/123` → `<root>/ABC_123` (sanitized)
  - `../../etc/passwd` → sanitized, still under root
  - Non-directory file at path → error
  - 5 path traversal scenario tests
- **Deps:** T-P101

### T-P301: Hook execution engine

- **Files:** `internal/workspace/hooks.go`
- **Spec:** Section 12.4, 6.3.4
- **Work:**
  - `RunHook(ctx, name, script, workspaceDir, timeoutMs)` via `bash -lc <script>` with cwd = workspace
  - Timeout via `context.WithTimeout`, kills process group
  - Capture stdout/stderr for logging (truncate in logs)
  - Return error on non-zero exit or timeout
- **Acceptance:**
  - `echo hello` succeeds
  - `exit 1` returns error
  - `sleep 999` with 100ms timeout → killed, timeout error
  - Correct cwd, stdout/stderr captured
- **Deps:** T-P100

### T-P302: Workspace Prepare lifecycle

- **Files:** `internal/workspace/manager.go` (expand)
- **Spec:** Section 12.2, 12.3, 12.4
- **Work:**
  - `Prepare(ctx, issue, hookConfig)` steps:
    1. Resolve path (T-P300)
    2. Ensure directory exists
    3. If newly created AND `after_create` hook → run hook. Failure → remove partial dir, return error
    4. If `before_run` hook → run hook. Failure → return error (abort attempt)
  - Return `domain.Workspace{Path, WorkspaceKey, CreatedNow}`
  - Validate cwd == workspace_path before returning
- **Acceptance:**
  - New workspace → after_create → before_run → success
  - Existing workspace → after_create skipped → before_run → success
  - after_create failure → directory removed, error returned
  - before_run failure → error, directory preserved
  - Integration test with real filesystem
- **Deps:** T-P300, T-P301

### T-P303: Workspace cleanup (terminal + manual)

- **Files:** `internal/workspace/cleanup.go`
- **Spec:** Section 10.8, 12.5, 18.1
- **Work:**
  - `CleanForIssue(ctx, identifier)`: resolve path, run before_remove hook (failure logged/ignored), `os.RemoveAll` if under root
  - `CleanTerminal(ctx, terminalIssues)`: iterate, call CleanForIssue, log each, continue on individual failures
- **Acceptance:**
  - Removes existing workspace
  - No-op if doesn't exist
  - before_remove hook runs before removal
  - before_remove failure doesn't prevent removal
  - Never removes paths outside root
- **Deps:** T-P300, T-P301

### T-P304: `tempad clean` CLI commands

- **Files:** `cmd/tempad/clean.go`
- **Spec:** Section 18.1
- **Work:**
  - `tempad clean` — query tracker for terminal-state issues, remove matching workspaces
  - `tempad clean <identifier>` — remove workspace for specific issue (no tracker needed)
- **Acceptance:**
  - `tempad clean ABC-123` removes workspace
  - `tempad clean` with tracker access removes terminal workspaces
  - `tempad clean` without tracker → helpful error
  - Confirmation message for each removal
- **Deps:** T-P303, T-P204 (for tracker-based clean)

### Phase 3 Internal Dependency Chain

```text
{T-P300, T-P301} → T-P302 → T-P303 → T-P304
```

---

## 9. Phase 4: TUI Mode — PENDING

**Goal:** `tempad` (default, no flags) shows a live task board, lets developer select → claim → workspace → IDE.

**Prerequisites:** Phase 2 (tracker client), Phase 3 (workspace manager)

### T-P400: Claim mechanism (shared by TUI + daemon)

- **Files:** `internal/claim/claimer.go`
- **Spec:** Section 5.1, 5.2, 5.3
- **Work:**
  - `Claim(ctx, tracker, issueID, identity)`: assign → fetch → verify assignee. If mismatch → unassign → `ClaimConflictError`
  - `Release(ctx, tracker, issueID)`: unassign
  - Stateless — all state managed by caller
- **Acceptance:**
  - Successful claim assigns and verifies
  - Race lost → unassigns, returns conflict error
  - Tracker error in step 1 → returns error without step 2
  - Unit tests with mock tracker
- **Deps:** T-P200

### T-P401: Bubble Tea app model and message types

- **Files:** `internal/tui/app.go`, `internal/tui/messages.go`
- **Spec:** Section 9.1
- **Work:**
  - `Model` struct implementing `tea.Model` with config, tracker, workspace, claimer refs; task list, cursor, view state
  - Message types: `PollResultMsg`, `ClaimResultMsg`, `WorkspaceReadyMsg`, `IDEOpenedMsg`, `ConfigReloadMsg`, `tickMsg`
  - `Init()` returns `tea.Batch(pollCmd, tickCmd)`
- **Deps:** T-P105, T-P200, T-P300

### T-P402: Task board view — rendering

- **Files:** `internal/tui/board.go`, `internal/tui/styles.go`
- **Spec:** Section 9.2
- **Work:**
  - "Available Tasks" section (unassigned, active-state) and "My Active Tasks" section (assigned to current user)
  - Each row: identifier, title, priority indicator (P1/P2/P3/P4), state, labels
  - Sorting: priority asc (null last) → created_at oldest → identifier lexicographic
  - Blocked issues (Todo state + non-terminal blockers) shown with `[BLOCKED]` marker
  - Lip Gloss styles for selected row, priority colors, blocked dimming, section headers
  - Footer with keybinding hints
- **Deps:** T-P401

### T-P403: Task board — keyboard navigation and actions

- **Files:** `internal/tui/keys.go`, `internal/tui/app.go` (Update method)
- **Spec:** Section 9.5
- **Work:**
  - Key bindings: `j`/`↓` down, `k`/`↑` up, `Enter` select, `r` refresh, `d` details, `o` open URL, `u` release, `q`/`Ctrl+C` quit
  - Selection state preserved across refresh (match by issue ID)
- **Deps:** T-P402

### T-P404: Task detail view

- **Files:** `internal/tui/detail.go`
- **Spec:** Section 9.5
- **Work:**
  - Full-screen view: identifier, title, state, priority, description (wrapped), labels, blockers (with identifiers/states), URL, timestamps
  - `Esc`/`Backspace` returns to board, scrollable
- **Deps:** T-P401

### T-P405: Poll loop and live refresh

- **Files:** `internal/tui/app.go` (expand)
- **Spec:** Section 9.4
- **Work:**
  - `pollCmd()` calls `tracker.FetchCandidateIssues()` → `PollResultMsg`
  - `tickCmd` via `tea.Tick(pollInterval, …)`, `r` key for immediate refresh
  - "Refreshing..." indicator, error shown inline (don't crash)
  - No duplicate concurrent polls
- **Deps:** T-P403

### T-P406: Task selection flow — claim → workspace → IDE

- **Files:** `internal/tui/app.go` (expand)
- **Spec:** Section 9.3
- **Work:**
  - On Enter: "Claiming..." → `claim.Claim()` → on success: `workspace.Prepare()` → on ready: `bash -lc "<ide.command> <ide.args> <path>"` → "Opened in IDE" → return to board
  - Claim failure → error message, return to board
  - Workspace failure → error message, board still usable
  - Disable selection while claim in progress
- **Deps:** T-P400, T-P302, T-P405

### T-P407: Release claimed task from TUI

- **Files:** `internal/tui/app.go` (expand)
- **Spec:** Section 5.3, 9.5
- **Work:**
  - `u` key on "My Active Tasks" item → confirm → `claim.Release()` → refresh
  - Only works on issues assigned to current user
- **Deps:** T-P400, T-P405

### T-P408: TUI mode entry point

- **Files:** `cmd/tempad/main.go` (expand root command)
- **Spec:** Section 9.1
- **Work:**
  - No `--daemon` flag → TUI mode: load+merge+validate config → create tracker → create workspace manager → startup terminal cleanup → `tea.Program` → `p.Run()`
  - Graceful exit on quit
- **Acceptance:**
  - `tempad` launches TUI with task board from Linear
  - Ctrl+C exits cleanly
  - Startup validation failure → exit 1
- **Deps:** T-P406, T-P407, T-P303

### Phase 4 Internal Dependency Chain

```text
T-P400 → T-P401 → {T-P402, T-P404} → T-P403 → T-P405 → T-P406 → T-P407 → T-P408
```

---

## 10. Phase 5: Daemon Mode Orchestrator — PENDING

**Goal:** `tempad --daemon` runs fully autonomous: poll → claim → dispatch → monitor → retry → reconcile.

**Prerequisites:** Phase 2 (tracker client), Phase 3 (workspace manager)

### T-P500: Orchestrator runtime state

- **Files:** `internal/orchestrator/orchestrator.go`
- **Spec:** Section 4.1.8, 10.2.1
- **Work:**
  - `Orchestrator` struct with state, tracker, workspace, agent, claimer, promptBuilder, config
  - Channels: `workerResults chan WorkerResult`, `retryTimers chan RetrySignal`, `configReload chan *ServiceConfig`
  - `WorkerResult` struct: issueID, exitCode, duration, error
  - `RetrySignal` struct: issueID, attempt, error, isContinuation
- **Deps:** T-P101, T-P200, T-P300, T-P400

### T-P501: Orchestrator main select loop

- **Files:** `internal/orchestrator/orchestrator.go` (expand)
- **Spec:** Section 10.3, Architecture Section 5.3
- **Work:**
  - `Run(ctx)`: startup cleanup → immediate tick → select loop over ctx.Done() / ticker.C / workerResults / retryTimers / configReload
  - Graceful shutdown: cancel workers, wait (with timeout), release all claims, log summary
  - Signal handling: SIGINT/SIGTERM → cancel context
- **Acceptance:**
  - Loop responds to all channel events
  - Graceful shutdown releases claims
  - No goroutine leaks after shutdown
- **Deps:** T-P500

### T-P502: Candidate selection and sorting

- **Files:** `internal/orchestrator/dispatch.go`
- **Spec:** Section 10.4
- **Work:**
  - `selectCandidates(issues, state)`: filter by id/identifier/title/state present, active states, not terminal, unassigned or self-assigned, not in running/claimed/retry, blocker rule (Todo state + non-terminal blockers → skip)
  - Sort: priority asc (null last) → created_at oldest → identifier lexicographic
- **Acceptance:**
  - Filters out running/claimed/retrying
  - Filters out blocked Todo issues
  - Includes self-assigned (resumption)
  - Correct sort order
  - 10+ candidate unit tests
- **Deps:** T-P101

### T-P503: Concurrency control

- **Files:** `internal/orchestrator/dispatch.go` (expand)
- **Spec:** Section 10.5
- **Work:**
  - `availableSlots(state)` → `max(max_concurrent - running_count, 0)`
  - `stateSlotAvailable(state, issueState)`: normalize state, check `max_concurrent_by_state`, invalid entries (non-positive) ignored
- **Acceptance:**
  - Global: 5 running, max=5 → 0 slots
  - Per-state: 2 "todo" running, limit=2 → no more todo
  - Invalid per-state (-1) → ignored, global fallback
- **Deps:** T-P502

### T-P504: Dispatch loop — claim and spawn workers

- **Files:** `internal/orchestrator/dispatch.go` (expand)
- **Spec:** Section 10.3 (step 5)
- **Work:**
  - For each candidate while slots available: claim → add to claimed → spawn worker → add to running
  - Claim failure → log, continue next. Spawn failure → unassign, schedule retry
- **Deps:** T-P503, T-P400

### T-P505: Agent worker goroutine

- **Files:** `internal/orchestrator/worker.go`
- **Spec:** Section 19.5, 11.3, 11.4, 11.5
- **Work:**
  - `runWorker(ctx, issue, attempt, config)`: workspace.Prepare → prompt.Render → deliver prompt → set 7 env vars (TEMPAD_ISSUE_ID, TEMPAD_ISSUE_IDENTIFIER, TEMPAD_ISSUE_TITLE, TEMPAD_ISSUE_URL, TEMPAD_WORKSPACE, TEMPAD_ATTEMPT, TEMPAD_PROMPT_FILE) → agent.Launch → handle.Wait → after_run hook → send WorkerResult
  - Tee stdout/stderr to log file + track lastOutputAt for stall detection
  - Respect turn_timeout_ms via context deadline
- **Acceptance:**
  - Full lifecycle exercised
  - All 7 env vars set
  - stdout/stderr logged to per-issue file
  - turn_timeout kills agent
  - after_run runs even on failure
- **Deps:** T-P302, T-P108, T-P506

### T-P506: Prompt delivery (4 methods)

- **Files:** `internal/agent/delivery.go`
- **Spec:** Section 11.3, 6.3.5
- **Work:**
  - `DeliverPrompt(method, prompt, workspacePath)`:
    - `"file"` → write `<workspace>/PROMPT.md`, set env var, pass path as first arg
    - `"stdin"` → return `io.Reader` to pipe to stdin
    - `"arg"` → append prompt as CLI argument
    - `"env"` → set `TEMPAD_PROMPT` env var
- **Deps:** T-P100

### T-P507: Agent subprocess launcher

- **Files:** `internal/agent/launcher.go` (implement), `internal/agent/process.go`
- **Spec:** Section 11.1, 11.3, 11.4
- **Work:**
  - `SubprocessLauncher` implementing `agent.Launcher`
  - Build command: `bash -lc "<command> <args>"`, set cwd, env, handle prompt delivery
  - Return `RunHandle` with Wait (blocks → ExitResult), Cancel (SIGTERM → 5s → SIGKILL), Stdout/Stderr readers
- **Acceptance:**
  - `echo hello && exit 0` → exit code 0
  - `exit 1` → exit code 1
  - Cancel kills subprocess, correct cwd and env
- **Deps:** T-P506

### T-P508: Agent output handling and stall detection

- **Files:** `internal/agent/output.go`
- **Spec:** Section 11.4, 10.7 (Part A)
- **Work:**
  - `OutputMonitor`: tee to log file, update atomic `lastOutputAt` on each read, optionally parse JSON lines, tolerate no output
  - `LastOutputAt()` for stall detection
- **Deps:** T-P507

### T-P509: Worker exit handling

- **Files:** `internal/orchestrator/orchestrator.go` (handleWorkerExit)
- **Spec:** Section 19.6
- **Work:**
  - Remove from running. Exit 0 → add to completed, schedule continuation retry (1s, no retry count). Exit != 0 → schedule failure retry (exponential backoff).
- **Deps:** T-P501, T-P510

### T-P510: Retry scheduling and backoff calculation

- **Files:** `internal/orchestrator/retry.go`
- **Spec:** Section 10.6
- **Work:**
  - `scheduleRetry(state, issueID, attempt, opts)`: cancel existing timer, compute delay (continuation: 1000ms fixed; failure: `min(10000 * 2^(attempt-1), max_retry_backoff_ms)`), store RetryEntry, `time.AfterFunc`
  - `handleRetry(signal)`: pop entry, fetch issue, if not found/not active → release claim. If attempt > max_retries (failure only) → release, log. No slots → requeue. Else → dispatch.
  - Continuation retries do NOT count toward max_retries
- **Acceptance:**
  - Continuation: always 1s
  - Failure delays: 10s, 20s, 40s, 80s, 160s, 300s (capped)
  - Max retries (10) → claim released
  - Existing timer cancelled on new retry
  - 10-attempt backoff formula unit tests
- **Deps:** T-P501

### T-P511: Active run reconciliation

- **Files:** `internal/orchestrator/reconcile.go`
- **Spec:** Section 10.7
- **Work:**
  - Part A — Stall detection: check `lastOutputAt`, if exceeds stall_timeout → cancel worker, schedule retry. stall_timeout ≤ 0 → skip.
  - Part B — Tracker state refresh: fetch states for running IDs. Terminal → cancel + clean workspace. Active → update snapshot. Neither → cancel (no cleanup). Fetch failure → keep running, retry next tick.
- **Acceptance:**
  - Stalled agent cancelled and retried
  - Terminal → killed + cleaned, non-active → killed + preserved
  - Fetch failure → agents kept running
  - stall_timeout=0 → skipped
- **Deps:** T-P508, T-P303

### T-P512: Daemon mode entry point

- **Files:** `cmd/tempad/main.go` (expand for --daemon)
- **Spec:** Section 10.1, 18.1, 18.2
- **Work:**
  - `--daemon` flag → load+merge+validate (agent.command required) → create components → `orchestrator.Run(ctx)`
  - SIGINT/SIGTERM → cancel context. Exit 0 normal, non-zero startup failure.
- **Deps:** T-P501 through T-P511

### T-P513: Daemon mode integration test

- **Files:** `internal/orchestrator/integration_test.go`
- **Spec:** Section 20.1
- **Work:**
  - Mock tracker (3 issues), mock agent (`echo done && exit 0`)
  - Verify: claimed → workers spawned → continuation retry → terminal → workspace cleaned → max retries → claim released
  - Test graceful shutdown. Run with `go test -race`. goleak for goroutine leaks.
- **Deps:** T-P512

### Phase 5 Internal Dependency Chain

```text
T-P500 → T-P501 → {T-P502, T-P506, T-P507} → {T-P503, T-P508} → T-P504 → T-P505 → {T-P509, T-P510, T-P511} → T-P512 → T-P513
```

---

## 11. Phase 6: Hot Reload + Logging + Polish — PENDING

**Goal:** Dynamic config reload, structured logging, production-ready polish.

**Prerequisites:** Phase 4 (TUI), Phase 5 (Daemon)

### T-P600: WORKFLOW.md file watcher with debounce

- **Files:** `internal/config/watcher.go`
- **Spec:** Section 8.2
- **Work:**
  - `StartWatcher(path, reload chan<-)` using fsnotify
  - 500ms debounce on change events
  - On change: re-parse workflow → re-merge → validate. Valid → send on channel. Invalid → log error, keep last known good.
  - Handle rename-and-replace pattern (re-add watch)
  - In-flight agents unaffected
- **Acceptance:**
  - Edit WORKFLOW.md → new config within 1s
  - Rapid edits → debounced to single reload
  - Invalid edit → error logged, old config kept
  - File deleted and recreated → re-watched
- **Deps:** T-P105

### T-P601: Structured logging setup

- **Files:** `internal/logging/setup.go`, `internal/logging/rotate.go`
- **Spec:** Section 15.1, 15.2
- **Work:**
  - `Setup(cfg)` configuring slog: TUI mode → stderr (don't interfere with TUI), daemon → file sink (`~/.tempad/logs/tempad.log`)
  - Log level from config (debug/info/warn/error), stable key=value format
  - Context fields: issue_id, issue_identifier, mode, attempt, agent_pid
  - Rotation: size-based (50MB default, keep 5 rotated files)
  - Per-issue agent logs: `~/.tempad/logs/<identifier>/agent.log`
  - Create log directories automatically
- **Deps:** T-P105

### T-P602: Config reload integration with orchestrator

- **Files:** `internal/orchestrator/orchestrator.go` (configReload case)
- **Spec:** Section 8.2
- **Work:**
  - On configReload: update poll_interval (reset ticker), max_concurrent, backoff/timeout settings, prompt template
  - Log which fields changed. Do NOT restart in-flight agents.
- **Deps:** T-P600, T-P501

### T-P603: Config reload integration with TUI

- **Files:** `internal/tui/app.go` (ConfigReloadMsg)
- **Spec:** Section 8.2
- **Work:**
  - On ConfigReloadMsg: update config reference, reset poll interval, show "Config reloaded" status
  - Invalid reload → show error in status bar
- **Deps:** T-P600, T-P401

---

## 12. Phase 7: HTTP Server Extension — PENDING

**Goal:** Optional `--port` enables REST API and dashboard for daemon mode observability.

**Prerequisite:** Phase 5 (Daemon orchestrator)

### T-P700: HTTP server setup and lifecycle

- **Files:** `internal/server/server.go`
- **Spec:** Section 15.5
- **Work:**
  - `NewServer(port, orchestrator)` with Chi router
  - Bind `127.0.0.1:<port>` (loopback only)
  - Graceful shutdown on context cancellation
  - `port=0` for ephemeral in tests
  - 405 for unsupported methods, error envelope `{"error": {"code": "…", "message": "…"}}`
- **Deps:** T-P501

### T-P701: API endpoints

- **Files:** `internal/server/handlers.go`
- **Spec:** Section 15.5
- **Work:**
  - `GET /` — HTML dashboard (server-rendered)
  - `GET /api/v1/state` — JSON: running sessions, retry queue, aggregates, poll info
  - `GET /api/v1/<identifier>` — JSON: issue-specific runtime details. 404 if unknown.
  - `POST /api/v1/refresh` — Queue immediate poll, return 202 Accepted
  - Read state via thread-safe snapshot method
- **Deps:** T-P700

### T-P702: HTTP server CLI integration

- **Files:** `cmd/tempad/main.go` (expand)
- **Spec:** Section 15.5, 18.1
- **Work:**
  - `--port` flag: if port > 0 AND daemon → start server alongside orchestrator. TUI mode → ignore/warn.
  - Log bound address on startup
- **Acceptance:**
  - `tempad --daemon --port 8080` starts both
  - `curl localhost:8080/api/v1/state` returns JSON
- **Deps:** T-P701, T-P512

---

## 13. Phase 8: Testing + Hardening — PENDING

**Goal:** Full test coverage, race detection, goroutine leak prevention, production readiness.

**Prerequisites:** All previous phases

### T-P800: Unit test coverage for all packages

- **Files:** `*_test.go` across all packages
- **Spec:** Section 20.1
- **Focus areas:** Config merge (10+ cases), workflow parsing edges, sanitization/normalization, candidate selection/sorting, backoff formula (10 attempts), concurrency limits, hook timeout/failure semantics, prompt rendering with all template features
- **Acceptance:** `go test ./...` passes, all Spec 20.1 test cases covered
- **Deps:** All phases

### T-P801: Race condition detection

- **Files:** All test files
- **Work:** Run `go test -race ./...`, fix any races found, verify channel-only communication
- **Acceptance:** Zero race warnings
- **Deps:** T-P800

### T-P802: Goroutine leak detection

- **Files:** `internal/orchestrator/leak_test.go`
- **Work:** `go.uber.org/goleak` in orchestrator tests. Test: start, dispatch 3, shutdown → zero leaks. Cancel via reconciliation → zero leaks. Retry timer after shutdown → no panic.
- **Deps:** T-P800

### T-P803: End-to-end integration test

- **Files:** `test/e2e_test.go`
- **Spec:** Section 20.1, 20.3
- **Work:**
  - Temp WORKFLOW.md + mock tracker (httptest) + mock agent (`echo done`)
  - Verify: issues claimed, agents run, exit 0, continuation check, release
  - Verify: structured logs, workspace creation/cleanup
  - Test config reload mid-run and graceful shutdown
- **Deps:** All phases

### T-P804: Signal handling and graceful shutdown verification

- **Files:** `cmd/tempad/main_test.go`
- **Spec:** Section 18.2
- **Work:** SIGINT/SIGTERM → exit 0, all agents terminated, all claims released, shutdown within 30s
- **Deps:** T-P512

### T-P805: Real Linear smoke test suite

- **Files:** `test/smoke_test.go`
- **Spec:** Section 20.3
- **Work:**
  - Tagged `//go:build smoke`, requires real Linear credentials
  - Fetch candidates, claim/release test issue, verify normalization, run echo agent, clean up
  - Isolated test identifiers, cleanup always runs
- **Deps:** All phases

---

## 14. Spec Coverage Cross-Check

Every item from Spec Section 21.1 (Required for Conformance) is mapped to tickets:

| Spec Requirement | Ticket(s) |
| ----------------- | ----------- |
| CLI with TUI + daemon modes | T-P107, T-P408, T-P512 |
| WORKFLOW.md loader | T-P102 |
| User config | T-P103 |
| Merged config layer | T-P105 |
| Dynamic WORKFLOW.md reload | T-P600 |
| Tracker client (fetch + assign + state refresh) | T-P200 to T-P204 |
| Assignment-based claim with race detection | T-P400 |
| Workspace manager with sanitized paths | T-P300 to T-P302 |
| Workspace lifecycle hooks | T-P301, T-P302 |
| Agent-agnostic launcher | T-P507 |
| Prompt rendering with issue + attempt | T-P108 |
| Daemon orchestrator (poll, dispatch, reconcile, retry) | T-P501 to T-P511 |
| Structured logging | T-P601 |
| TUI: board, selection, claim, IDE | T-P401 to T-P408 |

---

## 15. Open Items

These are implementation details to resolve during coding, not architectural blockers:

1. **Linear user ID resolution.** `tracker.identity` can be email or user ID. Linear's assignment mutation needs a user ID. May need a `resolveIdentity(email) → userID` call at startup.

2. **Stall detection granularity.** Tracking "last agent output" requires a goroutine reading stdout/stderr pipes and updating an atomic `lastOutputAt` timestamp.

3. **TUI + daemon coexistence.** Mutually exclusive in v1. Future "TUI dashboard for daemon mode" is a natural extension — the architecture supports it via state snapshots.

4. **Liquid `default` filter.** Verified working with `osteele/liquid` in Phase 1 implementation.

5. **Agent output log files.** Each worker goroutine should tee stdout/stderr to `~/.tempad/logs/<issue_identifier>/agent.log` and the stall detection reader.

---

## 16. Parallelization Opportunities

```text
Phase 1: T-P100 → {T-P101, T-P102, T-P103, T-P104} → T-P105 → T-P106 → T-P107 → T-P108 → T-P109
Phase 2: {T-P200, T-P201} → T-P202 → T-P203 → T-P204 → T-P205
Phase 3: {T-P300, T-P301} → T-P302 → T-P303 → T-P304
Phase 4: T-P400 → T-P401 → {T-P402, T-P404} → T-P403 → T-P405 → T-P406 → T-P407 → T-P408
Phase 5: T-P500 → T-P501 → {T-P502, T-P506, T-P507} → {T-P503, T-P508} → T-P504 → T-P505 → {T-P509, T-P510, T-P511} → T-P512 → T-P513
```

**Cross-phase parallelism:** Phases 2 and 3 can run in parallel. Phases 4 and 5 can run in parallel (both need 2+3). Phase 7 only needs Phase 5.

---

*Generated from SPEC_v1.md (v1.0.0), ARCHITECTURE_GO_v1.md (v1.0.0), STACK_COMPARISON_v1.md (v1.0.0), BACKLOG_v1.md (v1.0.0)*
*Module: github.com/oneneural/tempad | Go 1.22+ | 57 tickets across 8 phases*
