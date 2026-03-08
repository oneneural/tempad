# TEMPAD Implementation Backlog

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Module** | `github.com/oneneural/tempad` |
| **Date** | 2026-03-08 |
| **Derived from** | `SPEC_v1.md` (v1.0.0), `ARCHITECTURE_GO_v1.md` (v1.0.0) |

**Status: All 57/57 tickets complete across all 8 phases.**

Each ticket has a unique ID (`T-PXXX`), maps to a spec section, lists its file targets, acceptance criteria, and dependencies. Tickets within a phase can be worked in dependency order — independent tickets are marked parallelizable.

---

## Phase 1: Foundation (CLI + Config + Workflow Loader + Domain)

**Goal:** `tempad init` and `tempad validate` work. All domain types defined. Config loads, merges, validates.

### T-P100: Initialize Go module and project scaffold

- **Files:** `code/go/go.mod`, `code/go/cmd/tempad/main.go`, `code/go/internal/` directory tree
- **Spec:** Section 18.1
- **Work:**
  - `go mod init github.com/oneneural/tempad`
  - Create directory tree matching architecture doc Section 3
  - Stub `main.go` with Cobra root command
  - Add `.gitignore` for Go binaries
- **Acceptance:**
  - `go build ./cmd/tempad` compiles
  - `./tempad --help` prints usage
- **Deps:** None (start here)
- **Parallelizable:** Yes (with T-P101)

---

### T-P101: Define all domain model structs

- **Files:** `internal/domain/issue.go`, `internal/domain/workspace.go`, `internal/domain/run.go`, `internal/domain/state.go`
- **Spec:** Section 4.1 (all subsections), Section 4.2
- **Work:**
  - `Issue` struct with all 14 fields from Spec 4.1.1 (id, identifier, title, description, priority, state, assignee, branch_name, url, labels, blocked_by, created_at, updated_at)
  - `BlockerRef` struct (id, identifier, state)
  - `WorkflowDefinition` struct (config map, prompt_template string) — Spec 4.1.2
  - `Workspace` struct (path, workspace_key, created_now) — Spec 4.1.5
  - `RunAttempt` struct (issue_id, issue_identifier, attempt, workspace_path, started_at, status, error) — Spec 4.1.6
  - `RetryEntry` struct (issue_id, identifier, attempt, due_at_ms, timer_handle, error) — Spec 4.1.7
  - `OrchestratorState` struct (poll_interval_ms, max_concurrent_agents, running, claimed, retry_attempts, completed, agent_totals) — Spec 4.1.8
  - `SanitizeIdentifier(identifier string) string` utility — replace `[^A-Za-z0-9._-]` with `_` per Spec 4.2
  - `NormalizeState(state string) string` utility — trim + lowercase per Spec 4.2
- **Acceptance:**
  - All structs compile
  - `SanitizeIdentifier("ABC-123/foo bar")` returns `"ABC-123_foo_bar"`
  - `NormalizeState("  In Progress  ")` returns `"in progress"`
  - Unit tests pass for sanitization and normalization
- **Deps:** T-P100
- **Parallelizable:** Yes (with T-P102, T-P103)

---

### T-P102: Workflow loader — parse WORKFLOW.md

- **Files:** `internal/config/workflow.go`
- **Spec:** Section 6.1, 6.2, 6.3, 6.5
- **Work:**
  - `LoadWorkflow(path string) (*domain.WorkflowDefinition, error)`
  - File discovery: explicit path or `./WORKFLOW.md` default
  - Parse `---` delimited YAML front matter from markdown body
  - If no front matter, treat entire file as prompt body with empty config map
  - YAML must decode to map — non-map is `workflow_front_matter_not_a_map` error
  - Trim prompt body whitespace
  - Unknown keys ignored (forward compat)
  - Error types: `missing_workflow_file`, `workflow_parse_error`, `workflow_front_matter_not_a_map`
- **Acceptance:**
  - Loads example WORKFLOW.md from Spec Appendix A correctly
  - Missing file returns `missing_workflow_file`
  - YAML array returns `workflow_front_matter_not_a_map`
  - No front matter → empty config, full file as prompt
  - Front matter with unknown keys loads without error
  - Unit tests for all 5 edge cases
- **Deps:** T-P100, T-P101
- **Parallelizable:** Yes (with T-P103)

---

### T-P103: User config loader

- **Files:** `internal/config/user.go`
- **Spec:** Section 7.1, 7.2, 7.3
- **Work:**
  - `LoadUserConfig(path string) (*UserConfig, error)`
  - Default path: `~/.tempad/config.yaml`
  - Parse YAML into `UserConfig` struct with fields:
    - `tracker.identity` (string)
    - `tracker.api_key` (string or `$VAR`)
    - `ide.command` (string, default `"code"`)
    - `ide.args` (string or null)
    - `agent.command` (string)
    - `agent.args` (string or null)
    - `display.theme` (string, default `"auto"`)
    - `logging.level` (string, default `"info"`)
    - `logging.file` (string, default `"~/.tempad/logs/tempad.log"`)
  - If file doesn't exist, return empty config (not an error — will be created by `tempad init`)
- **Acceptance:**
  - Loads example config from Spec Appendix B correctly
  - Missing file returns empty config without error
  - Malformed YAML returns parse error
  - `$VAR_NAME` values preserved as-is (resolution happens in merge step)
  - Unit tests
- **Deps:** T-P100
- **Parallelizable:** Yes (with T-P102)

---

### T-P104: Environment variable resolution

- **Files:** `internal/config/config.go` (or separate `resolve.go`)
- **Spec:** Section 8.1 (item 4), Section 6.3.1 (api_key `$VAR`)
- **Work:**
  - `ResolveEnvVar(value string) string` — if value starts with `$`, look up `os.Getenv`, return resolved value
  - If `$VAR_NAME` resolves to empty string, treat as missing (return `""`)
  - Apply to `tracker.api_key`, `workspace.root` fields
  - `~` expansion for paths (`workspace.root`, `logging.file`)
- **Acceptance:**
  - `$LINEAR_API_KEY` resolves when env var set
  - `$NONEXISTENT` resolves to `""` (treated as missing)
  - `~/workspaces` expands to absolute home path
  - Unit tests with controlled env vars
- **Deps:** T-P100
- **Parallelizable:** Yes

---

### T-P105: ServiceConfig struct and merge logic

- **Files:** `internal/config/config.go`, `internal/config/loader.go`
- **Spec:** Section 8.1, 8.4, Section 4.1.4
- **Work:**
  - Define `ServiceConfig` struct with all fields from Architecture doc Section 7.2 (33 fields)
  - Define `CLIFlags` struct for command-line overrides
  - `Merge(cliFlags, userConfig, workflowConfig) *ServiceConfig`
  - Merge precedence: CLI > User > Repo > EnvVar > Defaults
  - Personal fields (identity, api_key, ide, agent command) → user wins
  - Team fields (hooks, states, workspace root, concurrency) → repo wins
  - Apply `$VAR` resolution after merge
  - Apply all built-in defaults from Spec 8.4:
    - `tracker.endpoint` → `https://api.linear.app/graphql`
    - `tracker.active_states` → `["Todo", "In Progress"]`
    - `tracker.terminal_states` → `["Closed", "Cancelled", "Canceled", "Duplicate", "Done"]`
    - `polling.interval_ms` → `30000`
    - `workspace.root` → `<os.TempDir()>/tempad_workspaces`
    - `hooks.timeout_ms` → `60000`
    - `agent.prompt_delivery` → `"file"`
    - `agent.max_concurrent` → `5`
    - `agent.max_turns` → `20`
    - `agent.max_retries` → `10`
    - `agent.max_retry_backoff_ms` → `300000`
    - `agent.turn_timeout_ms` → `3600000`
    - `agent.stall_timeout_ms` → `300000`
    - `agent.read_timeout_ms` → `5000`
    - `ide.command` → `"code"`
    - `display.theme` → `"auto"`
    - `logging.level` → `"info"`
  - Handle `active_states`/`terminal_states` as list OR comma-separated string
- **Acceptance:**
  - CLI `--identity` overrides user config `tracker.identity`
  - User `agent.command` overrides repo `agent.command`
  - Repo `hooks.after_create` is NOT overridable by user config
  - All defaults applied when both configs are empty
  - Comma-separated `"Todo, In Progress"` parsed to `["Todo", "In Progress"]`
  - 10+ unit tests for merge precedence edge cases
- **Deps:** T-P102, T-P103, T-P104

---

### T-P106: Dispatch preflight validation

- **Files:** `internal/config/validation.go`
- **Spec:** Section 8.3
- **Work:**
  - `ValidateForStartup(cfg *ServiceConfig, mode string) error`
  - `ValidateForDispatch(cfg *ServiceConfig, mode string) error`
  - Checks (both startup and dispatch):
    - `tracker.kind` is present and is `"linear"`
    - `tracker.api_key` is present and non-empty (after `$` resolution)
    - `tracker.project_slug` is present (when kind=linear)
    - `tracker.identity` is present
  - Additional startup checks:
    - `agent.command` is present (daemon mode only)
    - Workflow file is loadable
  - Return structured errors with human-readable messages
- **Acceptance:**
  - Missing `tracker.kind` → clear error message naming the field
  - Missing `agent.command` in daemon mode → error
  - Missing `agent.command` in TUI mode → no error
  - Empty `api_key` after `$VAR` resolution → error says "api_key resolved to empty"
  - All 6 validation checks tested individually
- **Deps:** T-P105

---

### T-P107: Cobra CLI skeleton with `init` and `validate` commands

- **Files:** `cmd/tempad/main.go` (expand), `cmd/tempad/init.go`, `cmd/tempad/validate.go`
- **Spec:** Section 18.1, 18.2
- **Work:**
  - Root command: `tempad` (default → TUI mode, placeholder for now)
  - Flags on root: `--daemon`, `--workflow <path>`, `--port <port>`, `--identity <identity>`, `--agent <command>`, `--ide <command>`, `--log-level <level>`
  - `tempad init` subcommand:
    - Creates `~/.tempad/` directory if absent
    - Writes `~/.tempad/config.yaml` with commented defaults (use Spec Appendix B as template)
    - If file exists, print message and do not overwrite
  - `tempad validate` subcommand:
    - Loads workflow + user config
    - Merges
    - Runs validation
    - Prints "Configuration valid" or detailed errors
    - Exit 0 on valid, exit 1 on invalid
  - Exit behavior: exit 0 normal, non-zero startup failure
- **Acceptance:**
  - `tempad init` creates config file with all documented fields
  - `tempad init` again says "config already exists"
  - `tempad validate` with valid WORKFLOW.md + config → exit 0, prints "valid"
  - `tempad validate` with missing api_key → exit 1, prints error
  - `tempad --workflow /nonexistent` → exit 1, clear error
  - All flags parsed and available in CLIFlags struct
- **Deps:** T-P105, T-P106

---

### T-P108: Prompt builder with Liquid templates

- **Files:** `internal/prompt/builder.go`
- **Spec:** Section 6.4, Section 14 (all subsections)
- **Work:**
  - `Render(templateStr string, issue domain.Issue, attempt *int) (string, error)`
  - Use `github.com/osteele/liquid` engine
  - Enable strict variable checking (unknown vars → error)
  - Convert `domain.Issue` to `map[string]any` via `issueToMap()`:
    - All 14 fields accessible as `issue.id`, `issue.identifier`, etc.
    - `issue.labels` as list of strings (for `{% for %}`)
    - `issue.blocked_by` as list of maps with `id`, `identifier`, `state`
    - `issue.priority` as integer or nil
    - `issue.created_at`, `issue.updated_at` as ISO-8601 strings
  - Template input variables: `issue` (object), `attempt` (int or nil)
  - Error types: `template_parse_error`, `template_render_error`
  - If prompt body empty, use minimal default: `"Work on issue {{ issue.identifier }}: {{ issue.title }}"`
  - Verify `default` filter works (`{{ issue.priority | default: "None" }}`)
  - Verify `downcase` filter works (used in hook examples)
  - Verify `size` property works on arrays (`issue.blocked_by.size`)
- **Acceptance:**
  - Renders example template from Spec Appendix A correctly
  - Unknown variable `{{ issue.nonexistent }}` returns `template_render_error`
  - `attempt` is nil on first run → `{% if attempt %}` block skipped
  - `attempt` is 3 → retry context block rendered
  - `issue.labels` iterable in `{% for %}`
  - `default` filter works
  - Empty template → uses minimal default
  - 10+ unit tests covering all template features
- **Deps:** T-P101

---

### T-P109: Phase 1 integration test

- **Files:** `cmd/tempad/main_test.go` or `internal/config/integration_test.go`
- **Spec:** Section 20.1 (Config and Workflow), Section 18.1
- **Work:**
  - End-to-end test: create temp WORKFLOW.md + temp config.yaml
  - Run `tempad validate` programmatically
  - Verify correct merge, defaults, validation
  - Test `tempad init` creates file
  - Test prompt rendering with realistic template
- **Acceptance:**
  - Full config pipeline exercised
  - All Phase 1 components work together
  - Tests pass with `go test -race ./...`
- **Deps:** T-P107, T-P108

---

## Phase 2: Tracker Client (Linear)

**Goal:** All 6 tracker operations work against Linear's GraphQL API. Issues are normalized into domain model.

### T-P200: Tracker client interface and error types

- **Files:** `internal/tracker/client.go`, `internal/tracker/errors.go`
- **Spec:** Section 13.1, 13.4
- **Work:**
  - Define `Client` interface with 6 methods (from Architecture doc Section 4.1)
  - Define typed errors:
    - `UnsupportedTrackerKindError`
    - `MissingTrackerAPIKeyError`
    - `MissingTrackerProjectSlugError`
    - `MissingTrackerIdentityError`
    - `TrackerAPIRequestError` (transport)
    - `TrackerAPIStatusError` (non-200)
    - `TrackerAPIErrorsError` (GraphQL errors)
    - `TrackerClaimConflictError` (assignment race)
  - All errors implement Go `error` interface with structured messages
- **Acceptance:**
  - Interface compiles
  - Errors implement `error` and `errors.Is` / `errors.As` work
  - Error messages include relevant context (issue ID, HTTP status, etc.)
- **Deps:** T-P101
- **Parallelizable:** Yes (with T-P201)

---

### T-P201: Linear GraphQL query/mutation builders

- **Files:** `internal/tracker/linear/graphql.go`
- **Spec:** Section 13.2, Architecture doc Section 8.1
- **Work:**
  - Define GraphQL query strings as Go constants:
    - `candidateIssuesQuery` — filter by project slug, active states, unassigned. Pagination. Include all Issue fields: id, identifier, title, description, priority, state.name, assignee.id, assignee.email, branchName, url, labels.nodes.name, relations (for blockers), createdAt, updatedAt
    - `assignedToMeQuery` — same but filter assignee by current user (for resumption)
    - `issueStatesByIDsQuery` — batch node lookup, return id + state.name
    - `issuesByStatesQuery` — filter by state names (terminal cleanup)
    - `singleIssueQuery` — fetch one issue by ID (claim verification)
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

---

### T-P202: Linear HTTP transport and pagination

- **Files:** `internal/tracker/linear/client.go`, `internal/tracker/linear/pagination.go`
- **Spec:** Section 13.2
- **Work:**
  - `LinearClient` struct: httpClient, endpoint, apiKey, projectSlug, identity, timeout (30s)
  - `NewLinearClient(cfg *config.ServiceConfig) *LinearClient`
  - `do(ctx, query, vars, result)` — POST to endpoint, `Authorization: Bearer <key>`, JSON body, unmarshal, check errors
  - Cursor-based pagination helper:
    - `fetchAll[T](ctx, query, vars, extractPage func) ([]T, error)`
    - Loop until `pageInfo.hasNextPage` is false
    - Default page size: 50
  - HTTP timeout: 30000ms
  - Respect `context.Context` cancellation
- **Acceptance:**
  - Sends correct `Authorization` header
  - Paginates correctly (mock server with 3 pages)
  - Context cancellation aborts in-flight request
  - Non-200 response → `TrackerAPIStatusError`
  - Network error → `TrackerAPIRequestError`
  - Unit tests with httptest mock server
- **Deps:** T-P200, T-P201

---

### T-P203: Issue normalization (Linear → domain.Issue)

- **Files:** `internal/tracker/linear/normalize.go`
- **Spec:** Section 13.3, Section 4.1.1
- **Work:**
  - `normalizeIssue(raw linearIssueResponse) domain.Issue`
  - Field mapping:
    - `labels` → lowercase all label names
    - `blocked_by` → derive from inverse relations where type is `blocks`
    - `priority` → int only, non-int → nil
    - `assignee` → user ID or email
    - `created_at`, `updated_at` → parse ISO-8601
    - `branch_name` → from `branchName` field
  - Handle nil/missing fields gracefully
- **Acceptance:**
  - Labels `["Bug", "Frontend"]` → `["bug", "frontend"]`
  - Priority `2` → `2`, priority `"high"` → nil
  - Relation `{type: "blocks", relatedIssue: {id, identifier, state}}` → `blocked_by` entry
  - Missing fields don't panic
  - Unit tests with real Linear response fixture
- **Deps:** T-P201, T-P101

---

### T-P204: Implement all 6 tracker operations

- **Files:** `internal/tracker/linear/client.go` (expand)
- **Spec:** Section 13.1
- **Work:**
  1. `FetchCandidateIssues(ctx)` — unassigned + active states + my assigned (resumption). Merge and deduplicate.
  2. `FetchIssuesByStates(ctx, states)` — terminal cleanup query
  3. `FetchIssueStatesByIDs(ctx, ids)` — batch node lookup, return `map[id]state`
  4. `FetchIssue(ctx, id)` — single issue fetch
  5. `AssignIssue(ctx, issueID, identity)` — mutation
  6. `UnassignIssue(ctx, issueID)` — mutation with `assigneeId: null`
  - Identity resolution: if `identity` looks like email, resolve to Linear user ID at client construction time (cache it). Use `viewer` query or `users(filter: {email})` query.
- **Acceptance:**
  - Each operation handles success and error cases
  - `FetchCandidateIssues` returns normalized, deduplicated issues
  - `AssignIssue` sends correct mutation
  - `UnassignIssue` sets assignee to null
  - Identity resolution from email works
  - 6 unit tests (one per operation) with mock server
- **Deps:** T-P202, T-P203

---

### T-P205: Tracker client integration smoke test

- **Files:** `internal/tracker/linear/integration_test.go`
- **Spec:** Section 20.3
- **Work:**
  - Test tagged `//go:build integration`
  - Requires `LINEAR_API_KEY` and `LINEAR_TEST_PROJECT_SLUG` env vars
  - Fetch candidates from a real Linear project
  - Verify normalization works on real data
  - Assign/unassign a test issue (use a designated test issue)
  - Clean up after test
- **Acceptance:**
  - Test passes against real Linear API
  - No orphaned assignments left after test
  - Skips gracefully when env vars absent
- **Deps:** T-P204

---

## Phase 3: Workspace Manager + Hooks

**Goal:** Deterministic workspace creation, hook execution, safety invariants, cleanup.

### T-P300: Workspace path resolution and safety invariants

- **Files:** `internal/workspace/manager.go`
- **Spec:** Section 12.1, 12.2, 12.6
- **Work:**
  - `NewManager(workspaceRoot string) *Manager`
  - `resolvePath(identifier string) (string, error)`:
    - Sanitize identifier via `domain.SanitizeIdentifier()`
    - Compute: `filepath.Join(workspaceRoot, sanitizedKey)`
    - **Invariant 2:** Normalize both paths to absolute. Verify `workspacePath` has `workspaceRoot` as prefix. Reject path traversal (`..`).
  - `ensureDir(path string) (createdNow bool, error)`:
    - `os.MkdirAll(path, 0755)`
    - Detect if newly created vs already existed
    - If non-directory file exists at path, return error
- **Acceptance:**
  - `ABC-123` → `<root>/ABC-123`
  - `ABC/123` → `<root>/ABC_123` (sanitized)
  - `../../etc/passwd` → sanitized to `______etc_passwd`, still under root
  - Non-directory file at path → error
  - Unit tests for 5 path traversal scenarios
- **Deps:** T-P101

---

### T-P301: Hook execution engine

- **Files:** `internal/workspace/hooks.go`
- **Spec:** Section 12.4, Section 6.3.4
- **Work:**
  - `RunHook(ctx context.Context, name string, script string, workspaceDir string, timeoutMs int) error`
  - Execute via `bash -lc <script>` with `cwd` = workspace directory
  - Timeout via `context.WithTimeout`
  - Log hook start, completion, failure, timeout
  - Capture stdout/stderr for logging (truncate in logs)
  - Return error on non-zero exit or timeout
  - Timeout kills process group (not just parent)
- **Acceptance:**
  - `echo hello` runs and succeeds
  - `exit 1` returns error
  - `sleep 999` with 100ms timeout → killed, returns timeout error
  - Script runs with correct `cwd`
  - Stdout/stderr captured in logs
  - Unit tests
- **Deps:** T-P100

---

### T-P302: Workspace Prepare lifecycle

- **Files:** `internal/workspace/manager.go` (expand)
- **Spec:** Section 12.2, 12.3, 12.4
- **Work:**
  - `Prepare(ctx, issue, hookConfig) (*domain.Workspace, error)`
  - Steps:
    1. Resolve path (T-P300)
    2. Ensure directory exists
    3. If newly created AND `after_create` hook configured → run hook. If hook fails → remove partial directory, return error
    4. If `before_run` hook configured → run hook. If fails → return error (abort attempt)
  - Return `domain.Workspace{Path, WorkspaceKey, CreatedNow}`
  - **Invariant 1:** Validate cwd == workspace_path before returning
- **Acceptance:**
  - New workspace → `after_create` runs → `before_run` runs → success
  - Existing workspace → `after_create` skipped → `before_run` runs → success
  - `after_create` failure → directory removed, error returned
  - `before_run` failure → error returned, directory preserved
  - Integration test with actual filesystem
- **Deps:** T-P300, T-P301

---

### T-P303: Workspace cleanup (terminal + manual)

- **Files:** `internal/workspace/cleanup.go`
- **Spec:** Section 10.8, 12.5, 18.1 (clean command)
- **Work:**
  - `CleanForIssue(ctx, identifier string) error`:
    - Resolve path
    - Run `before_remove` hook if configured (failure logged, ignored)
    - `os.RemoveAll(path)` if path exists and is under root
  - `CleanTerminal(ctx, terminalIssues []domain.Issue) error`:
    - For each issue, call `CleanForIssue`
    - Log each removal
    - Continue on individual failures
- **Acceptance:**
  - Removes existing workspace directory
  - No-op if workspace doesn't exist
  - `before_remove` hook runs before removal
  - `before_remove` failure doesn't prevent removal
  - Never removes paths outside workspace root
  - Unit tests
- **Deps:** T-P300, T-P301

---

### T-P304: `tempad clean` CLI commands

- **Files:** `cmd/tempad/clean.go`
- **Spec:** Section 18.1
- **Work:**
  - `tempad clean` — query tracker for terminal-state issues, remove matching workspaces
  - `tempad clean <identifier>` — remove workspace for specific issue
  - Requires tracker client (needs API key) for `tempad clean` without args
  - `tempad clean <identifier>` works without tracker connection (just removes directory)
- **Acceptance:**
  - `tempad clean ABC-123` removes `<root>/ABC-123` workspace
  - `tempad clean` with tracker access removes terminal workspaces
  - `tempad clean` without tracker access → helpful error
  - Confirmation message for each removal
- **Deps:** T-P303, T-P204 (for tracker-based clean)

---

## Phase 4: TUI Mode

**Goal:** `tempad` (default, no flags) shows a live task board, lets developer select → claim → workspace → IDE.

### T-P400: Claim mechanism (shared by TUI + daemon)

- **Files:** `internal/claim/claimer.go`
- **Spec:** Section 5.1, 5.2, 5.3
- **Work:**
  - `Claim(ctx, tracker, issueID, identity) error`
    1. `tracker.AssignIssue(issueID, identity)`
    2. `tracker.FetchIssue(issueID)` — verify assignee
    3. If assignee != identity → `tracker.UnassignIssue(issueID)` → return `ClaimConflictError`
  - `Release(ctx, tracker, issueID) error`
    - `tracker.UnassignIssue(issueID)`
  - Stateless — all state managed by caller
- **Acceptance:**
  - Successful claim assigns and verifies
  - Race lost → unassigns and returns conflict error
  - Tracker error in step 1 → returns error without step 2
  - Tracker error in step 2 → returns error (leaves assignment, caller decides)
  - Unit tests with mock tracker
- **Deps:** T-P200

---

### T-P401: Bubble Tea app model and message types

- **Files:** `internal/tui/app.go`, `internal/tui/messages.go`
- **Spec:** Section 9.1
- **Work:**
  - Define `Model` struct implementing `tea.Model`:
    - Config, tracker client, workspace manager, claimer references
    - Task list (available + my active)
    - Selection cursor
    - Loading/error state
    - Current view (board / detail)
  - Define message types:
    - `PollResultMsg{issues []domain.Issue, err error}`
    - `ClaimResultMsg{issue domain.Issue, err error}`
    - `WorkspaceReadyMsg{workspace domain.Workspace, issue domain.Issue, err error}`
    - `IDEOpenedMsg{err error}`
    - `ConfigReloadMsg{cfg *config.ServiceConfig}`
    - `tickMsg{}`
  - `Init()` → return `tea.Batch(pollCmd, tickCmd)`
  - Tick interval from `config.PollIntervalMs`
- **Acceptance:**
  - Model compiles and implements `tea.Model`
  - Init returns poll + tick commands
  - All message types defined
- **Deps:** T-P105, T-P200, T-P300

---

### T-P402: Task board view — rendering

- **Files:** `internal/tui/board.go`, `internal/tui/styles.go`
- **Spec:** Section 9.2
- **Work:**
  - `View()` renders two sections:
    - "Available Tasks" — unassigned, active-state issues
    - "My Active Tasks" — issues assigned to current user
  - Each task row shows: identifier, title, priority indicator, state, labels
  - Priority indicators: `P1` `P2` `P3` `P4` (or color-coded)
  - Sorting: priority ascending (null last) → created_at oldest → identifier lexicographic
  - Blocked issues (Todo state + non-terminal blockers) shown with `[BLOCKED]` marker
  - Lip Gloss styles for: selected row, priority colors, blocked dimming, section headers
  - Header with project name and poll status
  - Footer with keybinding hints
- **Acceptance:**
  - Tasks rendered in correct sort order
  - Blocked tasks visually distinct
  - "My Active Tasks" section shows claimed issues
  - Empty state message when no tasks
  - Renders cleanly at 80-column terminal width
- **Deps:** T-P401

---

### T-P403: Task board — keyboard navigation and actions

- **Files:** `internal/tui/keys.go`, `internal/tui/app.go` (Update method)
- **Spec:** Section 9.5
- **Work:**
  - Key bindings:
    - `j`/`↓` — move cursor down
    - `k`/`↑` — move cursor up
    - `Enter` — select/pick task (trigger claim flow)
    - `r` — manual refresh (trigger poll)
    - `d` — view task details
    - `o` — open task URL in browser (`xdg-open` / `open`)
    - `u` — release claimed task (unassign)
    - `q`/`Ctrl+C` — quit
  - `Update()` handles each `KeyMsg` → dispatches appropriate `tea.Cmd`
  - Selection state preserved across refresh (match by issue ID)
- **Acceptance:**
  - All keybindings work
  - Cursor wraps or stops at list boundaries
  - Refresh doesn't reset cursor position
  - `q` exits cleanly
  - `o` opens URL (or shows error if no URL)
- **Deps:** T-P402

---

### T-P404: Task detail view

- **Files:** `internal/tui/detail.go`
- **Spec:** Section 9.5
- **Work:**
  - Full-screen view showing:
    - Identifier + title
    - State
    - Priority
    - Description (wrapped to terminal width)
    - Labels
    - Blockers (with identifiers and states)
    - URL
    - Created/updated timestamps
  - `Esc` or `Backspace` returns to board
  - Scrollable if content exceeds terminal height
- **Acceptance:**
  - All issue fields displayed
  - Long descriptions wrap correctly
  - Escape returns to board
  - Blockers listed with their states
- **Deps:** T-P401

---

### T-P405: Poll loop and live refresh

- **Files:** `internal/tui/app.go` (expand Update)
- **Spec:** Section 9.4
- **Work:**
  - `pollCmd()` — call `tracker.FetchCandidateIssues()`, return `PollResultMsg`
  - `tickCmd()` — `tea.Tick(pollInterval, func(t) tickMsg{})`
  - On `tickMsg` → dispatch `pollCmd`
  - On `PollResultMsg` → update task list, preserve selection
  - On `r` key → dispatch `pollCmd` immediately
  - Show "refreshing..." indicator during poll
  - Show error inline if poll fails (don't crash)
- **Acceptance:**
  - Task list updates every `polling.interval_ms`
  - Manual refresh works immediately
  - Selection preserved after refresh (matched by issue ID)
  - Poll error shown as status message, board still usable
  - No duplicate concurrent polls
- **Deps:** T-P403

---

### T-P406: Task selection flow — claim → workspace → IDE

- **Files:** `internal/tui/app.go` (expand Update)
- **Spec:** Section 9.3
- **Work:**
  - On `Enter`:
    1. Show "Claiming..." status
    2. `claimCmd(issue)` → `claim.Claim()` → `ClaimResultMsg`
    3. On success: `prepareWorkspaceCmd(issue)` → `workspace.Prepare()` → `WorkspaceReadyMsg`
    4. On workspace ready: `openIDECmd(workspace)` → exec `bash -lc "<ide.command> <ide.args> <path>"` → `IDEOpenedMsg`
    5. On IDE opened: show "Opened in IDE" status, return to board
  - On claim failure: show error message, return to board
  - On workspace failure: show error, return to board
  - Disable selection while claim is in progress
- **Acceptance:**
  - Full flow: select → claim → workspace → IDE opens
  - Claim race lost → "Someone else claimed this task" message
  - Workspace hook failure → error message, board still usable
  - IDE command executed with correct path
  - Can pick another task after IDE opens
- **Deps:** T-P400, T-P302, T-P405

---

### T-P407: Release claimed task from TUI

- **Files:** `internal/tui/app.go` (expand)
- **Spec:** Section 5.3, 9.5
- **Work:**
  - `u` key on a "My Active Tasks" item → confirm → `claim.Release()` → refresh
  - Only works on issues assigned to current user
  - Show confirmation prompt before releasing
- **Acceptance:**
  - Release unassigns issue
  - Issue moves from "My Active" to "Available" on next refresh
  - Cannot release someone else's task
- **Deps:** T-P400, T-P405

---

### T-P408: TUI mode entry point

- **Files:** `cmd/tempad/main.go` (expand root command)
- **Spec:** Section 9.1
- **Work:**
  - When no `--daemon` flag, run TUI mode:
    1. Load + merge + validate config
    2. Create tracker client
    3. Create workspace manager
    4. Run startup terminal workspace cleanup
    5. Create `tea.Program` with Model
    6. `p.Run()` — blocks until quit
  - Graceful exit on quit
- **Acceptance:**
  - `tempad` (no flags) launches TUI
  - TUI shows task board from Linear
  - Ctrl+C exits cleanly
  - Startup validation failure → exit 1 with error
- **Deps:** T-P406, T-P407, T-P303

---

## Phase 5: Daemon Mode Orchestrator

**Goal:** `tempad --daemon` runs fully autonomous: poll → claim → dispatch → monitor → retry → reconcile.

### T-P500: Orchestrator runtime state

- **Files:** `internal/orchestrator/orchestrator.go`
- **Spec:** Section 4.1.8, Section 10.2.1
- **Work:**
  - `Orchestrator` struct:
    - `state *domain.OrchestratorState`
    - `tracker tracker.Client`
    - `workspace workspace.Manager`
    - `agent agent.Launcher`
    - `claimer *claim.Claimer`
    - `promptBuilder *prompt.Builder`
    - `config *config.ServiceConfig`
    - Channels: `workerResults chan WorkerResult`, `retryTimers chan RetrySignal`, `configReload chan *config.ServiceConfig`
  - `WorkerResult` struct: issueID, exitCode, duration, error
  - `RetrySignal` struct: issueID, attempt, error, isContinuation
  - `NewOrchestrator(...)` constructor
- **Acceptance:**
  - Struct compiles with all required fields
  - Channels created with appropriate buffer sizes
- **Deps:** T-P101, T-P200, T-P300, T-P400

---

### T-P501: Orchestrator main select loop

- **Files:** `internal/orchestrator/orchestrator.go` (expand)
- **Spec:** Section 10.3, Architecture Section 5.3
- **Work:**
  - `Run(ctx context.Context) error`:
    1. Startup terminal workspace cleanup
    2. Schedule immediate tick
    3. Enter select loop:
       - `<-ctx.Done()` → graceful shutdown (cancel all workers, wait, exit)
       - `<-ticker.C` → `tick()`
       - `<-workerResults` → `handleWorkerExit(result)`
       - `<-retryTimers` → `handleRetry(signal)`
       - `<-configReload` → `applyNewConfig(cfg)`
  - Graceful shutdown:
    - Cancel context for all workers
    - Wait for all running workers to exit (with timeout)
    - Release all claims
    - Log shutdown summary
  - Signal handling: catch SIGINT/SIGTERM → cancel context
- **Acceptance:**
  - Loop runs and responds to all channel events
  - Graceful shutdown releases claims
  - SIGINT triggers clean shutdown
  - No goroutine leaks after shutdown
- **Deps:** T-P500

---

### T-P502: Candidate selection and sorting

- **Files:** `internal/orchestrator/dispatch.go`
- **Spec:** Section 10.4
- **Work:**
  - `selectCandidates(issues []domain.Issue, state *OrchestratorState) []domain.Issue`
  - Filter: each issue must have id, identifier, title, state
  - Filter: state in `active_states` AND NOT in `terminal_states`
  - Filter: unassigned OR assigned to current user
  - Filter: not in `state.running`, `state.claimed`, or `state.retry_attempts`
  - Filter: if state is "Todo" (normalized), skip if any `blocked_by` entry has non-terminal state
  - Sort: `priority` asc (null last) → `created_at` oldest → `identifier` lexicographic
- **Acceptance:**
  - Filters out running/claimed/retrying issues
  - Filters out Todo issues with non-terminal blockers
  - Includes issues assigned to self (resumption)
  - Sort order: P1 before P2, older before newer, ABC-1 before ABC-2
  - Null priority sorts last
  - Unit tests with 10+ candidates testing all filter/sort rules
- **Deps:** T-P101

---

### T-P503: Concurrency control

- **Files:** `internal/orchestrator/dispatch.go` (expand)
- **Spec:** Section 10.5
- **Work:**
  - `availableSlots(state) int` → `max(max_concurrent - len(state.running), 0)`
  - `stateSlotAvailable(state, issueState string) bool`:
    - Normalize issue state
    - Check `max_concurrent_by_state[normalized]` if present
    - Count running issues with that state
    - If per-state limit configured and count >= limit → false
    - Invalid entries (non-positive) ignored
    - If no per-state limit → global check only
- **Acceptance:**
  - Global limit: 5 running, max_concurrent=5 → 0 slots
  - Per-state: 2 "todo" running, max_concurrent_by_state["todo"]=2 → no more todo
  - Per-state with different states → independent limits
  - Invalid per-state entry (-1) → ignored, falls back to global
  - Unit tests
- **Deps:** T-P502

---

### T-P504: Dispatch loop — claim and spawn workers

- **Files:** `internal/orchestrator/dispatch.go` (expand)
- **Spec:** Section 10.3 (step 5), Section 19.4
- **Work:**
  - `dispatch(ctx, candidates, state) *OrchestratorState`:
    - For each candidate while `availableSlots > 0` and `stateSlotAvailable`:
      1. `claim.Claim(tracker, issue.ID, identity)` — if fails, skip to next
      2. Add to `state.claimed`
      3. Spawn worker goroutine (T-P505)
      4. Add to `state.running`
  - Claim failure → log and continue to next candidate
  - Spawn failure → `tracker.UnassignIssue()`, schedule retry
- **Acceptance:**
  - Dispatches up to available slots
  - Stops at global concurrency limit
  - Stops at per-state limit
  - Claim failure → skips issue, tries next
  - Spawn failure → releases claim, schedules retry
  - Unit tests with mock tracker + mock launcher
- **Deps:** T-P503, T-P400

---

### T-P505: Agent worker goroutine

- **Files:** `internal/orchestrator/worker.go`
- **Spec:** Section 19.5, Section 11.3, 11.4, 11.5
- **Work:**
  - `runWorker(ctx, issue, attempt, config) WorkerResult`:
    1. `workspace.Prepare(issue)` — if fails, return error result
    2. `prompt.Render(template, issue, attempt)` — if fails, return error result
    3. Deliver prompt per `prompt_delivery` method (T-P506)
    4. Set agent environment variables:
       - `TEMPAD_ISSUE_ID`, `TEMPAD_ISSUE_IDENTIFIER`, `TEMPAD_ISSUE_TITLE`, `TEMPAD_ISSUE_URL`, `TEMPAD_WORKSPACE`, `TEMPAD_ATTEMPT`, `TEMPAD_PROMPT_FILE`
    5. `agent.Launch(ctx, opts)` — if fails, run after_run hook, return error
    6. `handle.Wait()` — blocks until exit or context cancel
    7. Run `after_run` hook (best-effort, failure logged+ignored)
    8. Send `WorkerResult` on channel
  - Tee stdout/stderr to log file (`~/.tempad/logs/<identifier>/agent.log`) and track `lastOutputAt` for stall detection
  - Respect `turn_timeout_ms` via context deadline
- **Acceptance:**
  - Full lifecycle: workspace → prompt → deliver → launch → wait → after_run → result
  - All 7 env vars set correctly
  - stdout/stderr logged to per-issue file
  - turn_timeout_ms kills agent
  - Context cancellation (from reconciliation) kills agent
  - after_run hook runs even on failure
  - Unit tests with mock agent (shell script)
- **Deps:** T-P302, T-P108, T-P506

---

### T-P506: Prompt delivery (4 methods)

- **Files:** `internal/agent/delivery.go`
- **Spec:** Section 11.3 (prompt delivery), Section 6.3.5
- **Work:**
  - `DeliverPrompt(method string, prompt string, workspacePath string) (*DeliveryResult, error)`
  - Methods:
    - `"file"` → write `<workspace>/PROMPT.md`, set `TEMPAD_PROMPT_FILE` env var, pass path as first arg
    - `"stdin"` → return `io.Reader` to pipe to subprocess stdin
    - `"arg"` → append prompt as CLI argument
    - `"env"` → set `TEMPAD_PROMPT` env var
  - `DeliveryResult` struct: stdinPipe, extraArgs, extraEnv
- **Acceptance:**
  - `file` creates PROMPT.md in workspace
  - `stdin` provides reader with prompt content
  - `arg` adds prompt to args
  - `env` adds TEMPAD_PROMPT to env
  - Unit tests for all 4 methods
- **Deps:** T-P100

---

### T-P507: Agent subprocess launcher

- **Files:** `internal/agent/launcher.go` (implement), `internal/agent/process.go`
- **Spec:** Section 11.1, 11.3, 11.4
- **Work:**
  - `SubprocessLauncher` implementing `agent.Launcher` interface
  - `Launch(ctx, opts)`:
    - Build command: `bash -lc "<command> <args>"`
    - Set working directory to `opts.WorkspacePath`
    - Set environment variables from `opts.Env`
    - Handle prompt delivery (stdin pipe, extra args, extra env)
    - Start process with `os/exec`
    - Return `RunHandle` with Wait, Cancel, Stdout, Stderr
  - `Wait()` → blocks, returns `ExitResult{ExitCode, Duration}`
  - `Cancel()` → send SIGTERM, wait 5s, send SIGKILL
  - Capture stdout/stderr as `io.Reader` (pipe)
- **Acceptance:**
  - Simple agent (`echo hello && exit 0`) → exit code 0
  - Failing agent (`exit 1`) → exit code 1
  - Cancel kills subprocess
  - Working directory correct
  - Env vars present in subprocess
  - Duration tracked accurately
  - Unit tests with real subprocesses
- **Deps:** T-P506

---

### T-P508: Agent output handling and stall detection

- **Files:** `internal/agent/output.go`
- **Spec:** Section 11.4, Section 10.7 (Part A)
- **Work:**
  - `OutputMonitor` that reads from agent stdout/stderr:
    - Tees to log file
    - Updates atomic `lastOutputAt` timestamp on each read
    - Optionally parses JSON lines for structured events (non-JSON lines treated as plain log)
    - Tolerates agents with no output
  - `LastOutputAt() time.Time` for stall detection
  - Structured event parsing (optional):
    - `event`, `timestamp`, `message`, `usage` fields
    - Non-JSON lines → plain log, no error
- **Acceptance:**
  - `lastOutputAt` updates on each stdout/stderr write
  - JSON lines parsed when valid
  - Non-JSON lines don't cause errors
  - Silent agent → `lastOutputAt` stays at launch time
  - Unit tests
- **Deps:** T-P507

---

### T-P509: Worker exit handling

- **Files:** `internal/orchestrator/orchestrator.go` (expand handleWorkerExit)
- **Spec:** Section 19.6
- **Work:**
  - `handleWorkerExit(result WorkerResult)`:
    - Remove from `state.running`
    - If exit code 0 (normal):
      - Add to `state.completed`
      - Schedule continuation retry: 1s delay, does NOT count toward max_retries
    - If exit code != 0 (failure):
      - Schedule failure retry with exponential backoff
- **Acceptance:**
  - Normal exit → continuation scheduled at 1s
  - Failure exit → backoff scheduled
  - Running map updated
  - Completed set updated on normal exit
  - Unit tests
- **Deps:** T-P501, T-P510

---

### T-P510: Retry scheduling and backoff calculation

- **Files:** `internal/orchestrator/retry.go`
- **Spec:** Section 10.6
- **Work:**
  - `scheduleRetry(state, issueID, attempt, opts)`:
    - Cancel existing timer for same issue
    - Compute delay:
      - Continuation: `1000ms` fixed
      - Failure: `min(10000 * 2^(attempt-1), max_retry_backoff_ms)`
    - Store `RetryEntry` in `state.retry_attempts`
    - `time.AfterFunc(delay)` → send `RetrySignal` on channel
  - `handleRetry(signal RetrySignal)`:
    - Pop `RetryEntry` from map
    - `tracker.FetchIssue(issueID)` — if not found or not active → release claim, remove from claimed
    - If attempt > `max_retries` (failure-driven only) → release claim, log exhaustion
    - If no slots → requeue with attempt+1
    - Else → dispatch
  - Continuation retries do NOT count toward max_retries
- **Acceptance:**
  - Continuation delay: always 1s
  - Failure delays: 10s, 20s, 40s, 80s, 160s, 300s (capped)
  - Max retries (10) → claim released
  - Issue no longer active → claim released
  - No slots → requeued
  - Existing timer cancelled when new retry scheduled
  - Unit tests for backoff formula (all 10 attempts)
- **Deps:** T-P501

---

### T-P511: Active run reconciliation

- **Files:** `internal/orchestrator/reconcile.go`
- **Spec:** Section 10.7
- **Work:**
  - `reconcile(ctx, state)`:
    - **Part A — Stall detection:**
      - For each running issue, check `outputMonitor.LastOutputAt()`
      - If `time.Since(lastOutput) > stall_timeout_ms` → cancel worker, schedule retry
      - If `stall_timeout_ms <= 0` → skip stall detection
    - **Part B — Tracker state refresh:**
      - `tracker.FetchIssueStatesByIDs(runningIDs)`
      - For each:
        - Terminal state → cancel worker, clean workspace
        - Still active → update in-memory state snapshot
        - Neither active nor terminal → cancel worker (no cleanup)
      - If fetch fails → keep agents running, log, retry next tick
- **Acceptance:**
  - Stalled agent cancelled and retried
  - Terminal issue → agent killed + workspace cleaned
  - Non-active issue → agent killed, workspace preserved
  - Tracker fetch failure → agents kept running
  - `stall_timeout_ms=0` → stall detection skipped
  - Unit tests for all 4 reconciliation outcomes
- **Deps:** T-P508, T-P303

---

### T-P512: Daemon mode entry point

- **Files:** `cmd/tempad/main.go` (expand for --daemon)
- **Spec:** Section 10.1, 18.1, 18.2
- **Work:**
  - When `--daemon` flag:
    1. Load + merge + validate config (including `agent.command` required)
    2. Create all components
    3. `orchestrator.Run(ctx)` — blocks until signal
  - Signal handling: SIGINT/SIGTERM → cancel context
  - Exit 0 on normal shutdown, non-zero on startup failure
- **Acceptance:**
  - `tempad --daemon` starts orchestrator
  - Ctrl+C shuts down gracefully
  - Missing `agent.command` → exit 1 with error
  - All claims released on shutdown
- **Deps:** T-P501 through T-P511

---

### T-P513: Daemon mode integration test

- **Files:** `internal/orchestrator/integration_test.go`
- **Spec:** Section 20.1 (Daemon Mode)
- **Work:**
  - Mock tracker serving 3 issues
  - Mock agent (shell script: `echo done && exit 0`)
  - Start orchestrator, verify:
    - Issues claimed
    - Workers spawned
    - After exit → continuation retry
    - After issue terminal → workspace cleaned
    - Max retries → claim released
  - Test graceful shutdown
  - Run with `go test -race`
- **Acceptance:**
  - Full lifecycle exercised with mocks
  - No race conditions
  - No goroutine leaks (goleak)
- **Deps:** T-P512

---

## Phase 6: Hot Reload + Logging + Polish

**Goal:** Dynamic config reload, structured logging, production-ready polish.

### T-P600: WORKFLOW.md file watcher with debounce

- **Files:** `internal/config/watcher.go`
- **Spec:** Section 8.2
- **Work:**
  - `StartWatcher(path string, reload chan<- *ServiceConfig) (stop func(), err error)`
  - Use `fsnotify` to watch file
  - 500ms debounce timer on change events
  - On change: re-parse workflow → re-merge with cached user config + CLI flags → validate
  - If valid → send new config on channel
  - If invalid → log error, keep last known good (do NOT send)
  - Handle rename-and-replace pattern (re-add watch after rename)
  - In-flight agents not affected — new config applies to future ticks only
- **Acceptance:**
  - Edit WORKFLOW.md → new config applied within 1s
  - Rapid edits (editor save) → debounced to single reload
  - Invalid edit → error logged, old config kept
  - File deleted and recreated → re-watched
  - Unit test with real filesystem
- **Deps:** T-P105

---

### T-P601: Structured logging setup

- **Files:** `internal/logging/setup.go`, `internal/logging/rotate.go`
- **Spec:** Section 15.1, 15.2
- **Work:**
  - `Setup(cfg *ServiceConfig) *slog.Logger`
  - Configure `slog` with:
    - TUI mode: stderr sink (don't interfere with TUI)
    - Daemon mode: file sink (`~/.tempad/logs/tempad.log`)
    - Log level from config (`debug`, `info`, `warn`, `error`)
    - Stable `key=value` format
  - Required context fields: `issue_id`, `issue_identifier`, `mode`
  - Daemon agent sessions: `attempt`, `agent_pid`
  - Log rotation: size-based (default 50MB, keep 5 rotated files)
  - Log sink failure → warning on remaining sinks, don't crash
  - Create `~/.tempad/logs/` directory if absent
  - Per-issue agent logs: `~/.tempad/logs/<identifier>/agent.log`
- **Acceptance:**
  - TUI mode logs to stderr
  - Daemon mode logs to file
  - Log level filtering works
  - Rotation triggers at size limit
  - Agent logs written to per-issue directory
  - Missing log directory created automatically
- **Deps:** T-P105

---

### T-P602: Config reload integration with orchestrator

- **Files:** `internal/orchestrator/orchestrator.go` (expand configReload case)
- **Spec:** Section 8.2
- **Work:**
  - On `configReload` channel event:
    - Update `state.poll_interval_ms` → reset ticker
    - Update `state.max_concurrent_agents`
    - Update backoff/timeout settings
    - Update prompt template (for future renders)
    - Log which fields changed
  - Do NOT restart in-flight agents
- **Acceptance:**
  - Change `polling.interval_ms` → tick interval changes
  - Change `max_concurrent` → new limit applied next dispatch
  - Change prompt template → next agent gets new prompt
  - In-flight agents continue unaffected
- **Deps:** T-P600, T-P501

---

### T-P603: Config reload integration with TUI

- **Files:** `internal/tui/app.go` (expand ConfigReloadMsg handling)
- **Spec:** Section 8.2
- **Work:**
  - On `ConfigReloadMsg`:
    - Update internal config reference
    - Update poll interval (reset tick timer)
    - Show brief "Config reloaded" status message
  - Invalid reload → show error in status bar
- **Acceptance:**
  - Config change reflected in TUI
  - Poll interval change takes effect
  - Error shown for invalid config
- **Deps:** T-P600, T-P401

---

## Phase 7: HTTP Server Extension

**Goal:** Optional `--port` enables REST API and dashboard for daemon mode observability.

### T-P700: HTTP server setup and lifecycle

- **Files:** `internal/server/server.go`
- **Spec:** Section 15.5
- **Work:**
  - `NewServer(port int, orchestrator *Orchestrator) *Server`
  - Chi router
  - Bind to `127.0.0.1:<port>` (loopback only)
  - Graceful shutdown on context cancellation
  - `port=0` for ephemeral port in tests
  - Unsupported methods → `405 Method Not Allowed`
  - Error envelope: `{"error": {"code": "...", "message": "..."}}`
- **Acceptance:**
  - Server starts on configured port
  - Binds loopback only
  - Graceful shutdown works
  - 405 for wrong methods
- **Deps:** T-P501

---

### T-P701: API endpoints

- **Files:** `internal/server/handlers.go`
- **Spec:** Section 15.5
- **Work:**
  - `GET /` — HTML dashboard (simple server-rendered page showing state)
  - `GET /api/v1/state` — JSON: running sessions, retry queue, aggregate totals, poll info
  - `GET /api/v1/<identifier>` — JSON: issue-specific runtime details. 404 if unknown.
  - `POST /api/v1/refresh` — Queue immediate poll cycle, return `202 Accepted`
  - Read orchestrator state via snapshot method (thread-safe read)
- **Acceptance:**
  - `/api/v1/state` returns correct JSON shape
  - `/api/v1/ABC-123` returns issue details when running
  - `/api/v1/UNKNOWN` returns 404
  - `/api/v1/refresh` triggers poll, returns 202
  - Dashboard renders in browser
- **Deps:** T-P700

---

### T-P702: HTTP server CLI integration

- **Files:** `cmd/tempad/main.go` (expand)
- **Spec:** Section 15.5, 18.1
- **Work:**
  - `--port <port>` flag on root command
  - If port > 0 AND daemon mode → start HTTP server alongside orchestrator
  - If port > 0 AND TUI mode → ignore (or warn)
  - Log bound address on startup
- **Acceptance:**
  - `tempad --daemon --port 8080` starts both
  - `curl localhost:8080/api/v1/state` returns JSON
  - TUI mode with `--port` → warning logged
- **Deps:** T-P701, T-P512

---

## Phase 8: Testing + Hardening

**Goal:** Full test coverage, race detection, goroutine leak prevention, production readiness.

### T-P800: Unit test coverage for all packages

- **Files:** `*_test.go` across all packages
- **Spec:** Section 20.1
- **Work:**
  - Ensure every package has tests
  - Target: all critical paths covered
  - Focus areas:
    - Config merge precedence (10+ cases)
    - Workflow parsing edge cases
    - Sanitization and normalization
    - Candidate selection and sorting
    - Backoff formula (all 10 attempts)
    - Concurrency limits
    - Hook timeout/failure semantics
    - Prompt rendering with all template features
- **Acceptance:**
  - `go test ./...` passes
  - All spec Section 20.1 test cases covered
- **Deps:** All previous phases

---

### T-P801: Race condition detection

- **Files:** All test files
- **Spec:** Architecture doc Section 13 (risk: data races)
- **Work:**
  - Run `go test -race ./...` across all packages
  - Fix any race conditions found
  - Ensure orchestrator state is never accessed from multiple goroutines
  - Verify channel-only communication pattern
- **Acceptance:**
  - `go test -race ./...` passes with zero warnings
- **Deps:** T-P800

---

### T-P802: Goroutine leak detection

- **Files:** `internal/orchestrator/leak_test.go`
- **Work:**
  - Add `go.uber.org/goleak` to orchestrator tests
  - Verify shutdown leaves no orphan goroutines
  - Test: start orchestrator, dispatch 3 workers, shutdown → zero leaks
  - Test: start orchestrator, cancel workers via reconciliation → zero leaks
  - Test: retry timer fires after shutdown → no panic
- **Acceptance:**
  - goleak reports zero leaks in all test scenarios
- **Deps:** T-P800

---

### T-P803: End-to-end integration test

- **Files:** `test/e2e_test.go`
- **Spec:** Section 20.1, 20.3
- **Work:**
  - Full end-to-end test:
    1. Create temp WORKFLOW.md with mock agent command (`echo done`)
    2. Start daemon with mock tracker (httptest server)
    3. Mock tracker returns 2 issues
    4. Verify: issues claimed, agents run, exit 0, continuation check, eventually release
    5. Verify structured logs written
    6. Verify workspace created and cleaned
  - Test config reload mid-run
  - Test graceful shutdown
- **Acceptance:**
  - Full lifecycle passes
  - Logs contain expected key=value entries
  - Workspaces managed correctly
  - No race conditions or leaks
- **Deps:** All phases

---

### T-P804: Signal handling and graceful shutdown verification

- **Files:** `cmd/tempad/main_test.go`
- **Spec:** Section 18.2
- **Work:**
  - Test: send SIGINT to running daemon → exits 0
  - Test: send SIGTERM → exits 0
  - Verify: all running agents terminated
  - Verify: all claims released
  - Verify: shutdown completes within 30s timeout
- **Acceptance:**
  - Both signals trigger clean shutdown
  - Exit code 0
  - No orphan processes
- **Deps:** T-P512

---

### T-P805: Real Linear smoke test suite

- **Files:** `test/smoke_test.go`
- **Spec:** Section 20.3
- **Work:**
  - Tagged `//go:build smoke`
  - Requires real Linear credentials
  - Tests:
    1. Fetch candidates from test project
    2. Claim and release a test issue
    3. Verify normalization on real data
    4. Run agent (echo) on real claimed issue
    5. Clean up all test state
  - Isolated test identifiers to avoid conflicts
- **Acceptance:**
  - Passes against real Linear API
  - No orphaned assignments
  - Cleanup always runs (even on test failure)
- **Deps:** All phases

---

## Summary

| Phase | Tickets | Key Deliverable |
| ------- | --------- | ---------------- |
| 1 — Foundation | T-P100 to T-P109 (10) | `tempad init`, `tempad validate`, config pipeline, prompt builder |
| 2 — Tracker | T-P200 to T-P205 (6) | Linear GraphQL client, all 6 operations, normalization |
| 3 — Workspace | T-P300 to T-P304 (5) | Workspace lifecycle, hooks, safety, `tempad clean` |
| 4 — TUI | T-P400 to T-P408 (9) | Interactive task board, claim flow, IDE launch |
| 5 — Daemon | T-P500 to T-P513 (14) | Full orchestrator: dispatch, retry, reconcile, agent lifecycle |
| 6 — Polish | T-P600 to T-P603 (4) | Hot reload, structured logging |
| 7 — HTTP | T-P700 to T-P702 (3) | REST API, dashboard |
| 8 — Hardening | T-P800 to T-P805 (6) | Tests, race detection, goroutine leaks, e2e, smoke |
| **Total** | **57 tickets** | |

### Parallelization Opportunities

```text
Phase 1: T-P100 → {T-P101, T-P102, T-P103, T-P104} → T-P105 → T-P106 → T-P107 → T-P108 → T-P109
Phase 2: {T-P200, T-P201} → T-P202 → T-P203 → T-P204 → T-P205
Phase 3: {T-P300, T-P301} → T-P302 → T-P303 → T-P304
Phase 4: T-P400 → T-P401 → {T-P402, T-P404} → T-P403 → T-P405 → T-P406 → T-P407 → T-P408
Phase 5: T-P500 → T-P501 → {T-P502, T-P506, T-P507} → {T-P503, T-P508} → T-P504 → T-P505 → {T-P509, T-P510, T-P511} → T-P512 → T-P513
```

### Spec Coverage Cross-Check

Every item from Spec Section 21.1 (Required for Conformance) is covered:

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
