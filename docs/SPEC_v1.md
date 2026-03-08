# T.E.M.P.A.D. Service Specification

**Temporal Execution & Management Poll-Agent Dispatcher**

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Status** | Draft (language-agnostic) |
| **Date** | 2026-03-08 |
| **Authors** | Subodh / OneNeural |

Purpose: Define a client-side service that bridges issue trackers and developer workflows — letting
developers pick tasks from a board, open their preferred IDE, and optionally dispatch coding agents
headlessly.

---

## 1. Problem Statement

T.E.M.P.A.D. is a developer-local service that continuously reads work from an issue tracker
(Linear in this specification version), presents available tasks to the developer, and either opens
an IDE session for the selected task or runs a coding agent headlessly in an isolated workspace.

There is no central server. Each developer runs T.E.M.P.A.D. on their own machine. The issue
tracker is the shared coordination layer — task assignment prevents duplicate work across the team.

The service solves five operational problems:

- It gives developers a live view of available work and lets them choose what to pick up.
- It turns issue execution into a repeatable workflow instead of manual scripts.
- It isolates agent execution in per-issue workspaces so agent commands run only inside per-issue
  workspace directories.
- It keeps the workflow policy in-repo (`WORKFLOW.md`) so teams version the agent prompt and runtime
  settings with their code.
- It provides enough observability to operate and debug concurrent agent runs.

Important boundaries:

- T.E.M.P.A.D. is a scheduler/runner and tracker reader that runs on the developer's machine.
- There is no central server, fleet management, or multi-node coordination.
- The issue tracker (Linear) is the coordination layer for the team — assignment is the claim
  mechanism.
- Ticket writes (state transitions, comments, PR links) are typically performed by the coding agent
  using tools available in the workflow/runtime environment.
- A successful run may end at a workflow-defined handoff state (for example `Human Review`), not
  necessarily `Done`.

## 2. Goals and Non-Goals

### 2.1 Goals

- Present available tasks from the issue tracker in an interactive TUI.
- Let developers pick tasks and open their preferred IDE — or run agents headlessly in daemon mode.
- Claim tasks via issue tracker assignment to prevent duplicate work across the team.
- In daemon mode, auto-dispatch work with bounded concurrency and full lifecycle management.
- Create deterministic per-issue workspaces and preserve them across runs.
- Stop active runs when issue state changes make them ineligible.
- Recover from transient failures with exponential backoff (daemon mode).
- Load runtime behavior from a repository-owned `WORKFLOW.md` contract.
- Expose operator-visible observability (at minimum structured logs).
- Support restart recovery without requiring a persistent database.
- Stay agent-agnostic — work with any coding agent (Claude Code, Codex, OpenCode, etc.).

### 2.2 Non-Goals

- Central server or multi-node coordination.
- Fleet management, worker routing, or centralized dispatch.
- Rich web UI or multi-tenant control plane.
- General-purpose workflow engine or distributed job scheduler.
- Built-in business logic for how to edit tickets, PRs, or comments. (That logic lives in the
  workflow prompt and agent tooling.)
- Mandating a specific coding agent or IDE.
- Mandating strong sandbox controls beyond what the coding agent and host OS provide.

## 3. System Overview

### 3.1 Operating Modes

T.E.M.P.A.D. runs in one of two modes:

**TUI Mode (Interactive)**

The default mode. The developer sees a live task board in their terminal:

1. T.E.M.P.A.D. polls the issue tracker and displays available (unassigned, active-state) tasks.
2. The developer selects a task from the list.
3. T.E.M.P.A.D. claims the task by assigning it to the developer's tracker identity.
4. T.E.M.P.A.D. creates/reuses the per-issue workspace, runs workspace hooks, and opens the
   developer's configured IDE with the workspace directory.
5. T.E.M.P.A.D.'s job is done for that task. The developer (and/or their agent running inside the
   IDE) handles the rest. The agent updates the issue tracker per the `WORKFLOW.md` instructions.
6. The TUI returns to the task list. The developer can pick another task in parallel.

**Daemon Mode (Headless)**

Activated with `--daemon`. Runs without human interaction:

1. T.E.M.P.A.D. polls the issue tracker for eligible tasks automatically.
2. Claims tasks by assigning them to the configured tracker identity.
3. Creates workspaces and launches the configured coding agent as a subprocess.
4. Manages the full agent lifecycle: monitors progress, detects completion/failure, retries with
   exponential backoff, enforces stall timeouts, respects concurrency limits.
5. When an agent finishes, T.E.M.P.A.D. re-checks the issue state and continues or releases the
   claim.

Daemon mode is for dedicated AI worker machines or developers who want fully autonomous execution.

### 3.2 Main Components

1. `Workflow Loader`
   - Reads `WORKFLOW.md`.
   - Parses YAML front matter and prompt body.
   - Returns `{config, prompt_template}`.

2. `Config Layer`
   - Merges repo-level config (`WORKFLOW.md` front matter) with user-level config
     (`~/.tempad/config.yaml`).
   - Exposes typed getters for all runtime values.
   - Applies defaults and environment variable indirection.
   - Performs validation before dispatch.

3. `Issue Tracker Client`
   - Fetches candidate issues (unassigned, active-state) for the configured project.
   - Fetches current states for specific issue IDs (reconciliation in daemon mode).
   - Fetches terminal-state issues during startup cleanup.
   - Claims issues by assigning them to the current user's tracker identity.
   - Normalizes tracker payloads into a stable issue model.

4. `Orchestrator`
   - In TUI mode: presents tasks, handles user selection, triggers workspace + IDE open.
   - In daemon mode: owns the poll tick, manages in-memory runtime state, decides which issues to
     dispatch/retry/stop/release, tracks session metrics and retry queues.

5. `Workspace Manager`
   - Maps issue identifiers to workspace paths.
   - Ensures per-issue workspace directories exist.
   - Runs workspace lifecycle hooks.
   - Cleans workspaces for terminal issues.

6. `Agent Launcher`
   - In TUI mode: opens the configured IDE with the workspace directory. The developer's agent runs
     inside the IDE.
   - In daemon mode: launches the configured agent command as a subprocess in the workspace
     directory. Streams agent output for observability. Detects completion, failure, and stalls.

7. `TUI` (TUI mode only)
   - Renders the live task board, available tasks, running tasks, and status.
   - Handles user interaction: task selection, manual refresh, status overview.

8. `Logging`
   - Emits structured runtime logs to one or more configured sinks.

### 3.3 Abstraction Levels

1. `Policy Layer` (repo-defined)
   - `WORKFLOW.md` prompt body.
   - Team-specific rules for ticket handling, validation, and handoff.

2. `Configuration Layer` (repo + user config)
   - Repo: `WORKFLOW.md` front matter (tracker, hooks, agent defaults, workspace).
   - User: `~/.tempad/config.yaml` (IDE preference, default agent command, API keys, tracker
     identity).

3. `Coordination Layer` (orchestrator)
   - TUI mode: task presentation and selection.
   - Daemon mode: polling loop, issue eligibility, concurrency, retries, reconciliation.

4. `Execution Layer` (workspace + agent/IDE)
   - Filesystem lifecycle, workspace preparation, IDE launch or agent subprocess.

5. `Integration Layer` (tracker adapter)
   - API calls, normalization, and claim operations for the issue tracker.
   - Designed for future adapter extensibility beyond Linear.

6. `Observability Layer` (logs + TUI status)
   - Operator visibility into orchestrator and agent behavior.

### 3.4 External Dependencies

- Issue tracker API (Linear for `tracker.kind: linear` in this specification version).
- Local filesystem for workspaces and logs.
- Optional workspace population tooling (for example Git CLI, if used via hooks).
- A coding agent executable (any agent that accepts a prompt and runs in a directory — daemon mode).
- An IDE or editor that can be opened via CLI (TUI mode).
- Host environment authentication for the issue tracker and coding agent.

## 4. Core Domain Model

### 4.1 Entities

#### 4.1.1 Issue

Normalized issue record used by orchestration, TUI display, prompt rendering, and observability.

Fields:

- `id` (string) — Stable tracker-internal ID.
- `identifier` (string) — Human-readable ticket key (example: `ABC-123`).
- `title` (string)
- `description` (string or null)
- `priority` (integer or null) — Lower numbers are higher priority in display/dispatch sorting.
- `state` (string) — Current tracker state name.
- `assignee` (string or null) — Tracker user ID or email of the current assignee.
- `branch_name` (string or null) — Tracker-provided branch metadata if available.
- `url` (string or null)
- `labels` (list of strings) — Normalized to lowercase.
- `blocked_by` (list of blocker refs)
  - Each blocker ref contains:
    - `id` (string or null)
    - `identifier` (string or null)
    - `state` (string or null)
- `created_at` (timestamp or null)
- `updated_at` (timestamp or null)

#### 4.1.2 Workflow Definition

Parsed `WORKFLOW.md` payload:

- `config` (map) — YAML front matter root object.
- `prompt_template` (string) — Markdown body after front matter, trimmed.

#### 4.1.3 User Config

Personal preferences stored in `~/.tempad/config.yaml`:

- `tracker.identity` (string) — The developer's tracker user ID, email, or display name. Used for
  assignment-based claiming.
- `tracker.api_key` (string or `$VAR`) — May override or supplement the repo-level key.
- `ide.command` (string) — Shell command to open the IDE. Default: `code` (VS Code). Examples:
  `cursor`, `zed`, `idea`, `webstorm`.
- `ide.args` (string or null) — Additional arguments passed to the IDE command.
- `agent.command` (string) — Default agent command for daemon mode. Example:
  `claude-code --auto`, `codex`, `opencode`.
- `agent.args` (string or null) — Additional arguments passed to the agent command.

#### 4.1.4 Service Config (Merged View)

Typed runtime values derived from `WORKFLOW.md` config + `~/.tempad/config.yaml` + environment
variables.

Merge precedence (highest wins):

1. CLI flags
2. User config (`~/.tempad/config.yaml`)
3. Repo config (`WORKFLOW.md` front matter)
4. Environment variable indirection (`$VAR_NAME`)
5. Built-in defaults

For fields that exist in both repo and user config:

- User config wins for personal preferences (IDE, agent command, tracker identity, API keys).
- Repo config wins for team-shared settings (hooks, prompt template, workspace root, active/terminal
  states, concurrency limits).

#### 4.1.5 Workspace

Filesystem workspace assigned to one issue identifier.

Fields (logical):

- `path` (absolute workspace path)
- `workspace_key` (sanitized issue identifier)
- `created_now` (boolean, used to gate `after_create` hook)

#### 4.1.6 Run Attempt (Daemon Mode)

One execution attempt for one issue.

Fields (logical):

- `issue_id`
- `issue_identifier`
- `attempt` (integer or null, `null` for first run, `>=1` for retries/continuation)
- `workspace_path`
- `started_at`
- `status`
- `error` (optional)

#### 4.1.7 Retry Entry (Daemon Mode)

Scheduled retry state for an issue.

Fields:

- `issue_id`
- `identifier` (best-effort human ID for logs)
- `attempt` (integer, 1-based for retry queue)
- `due_at_ms` (monotonic clock timestamp)
- `timer_handle` (runtime-specific timer reference)
- `error` (string or null)

#### 4.1.8 Orchestrator Runtime State (Daemon Mode)

Single authoritative in-memory state owned by the orchestrator.

Fields:

- `poll_interval_ms` (current effective poll interval)
- `max_concurrent_agents` (current effective global concurrency limit)
- `running` (map `issue_id -> running entry`)
- `claimed` (set of issue IDs reserved/running/retrying)
- `retry_attempts` (map `issue_id -> RetryEntry`)
- `completed` (set of issue IDs; bookkeeping only, not dispatch gating)
- `agent_totals` (aggregate tokens + runtime seconds, if the agent reports them)

### 4.2 Stable Identifiers and Normalization Rules

- `Issue ID` — Use for tracker lookups and internal map keys.
- `Issue Identifier` — Use for human-readable logs, TUI display, and workspace naming.
- `Workspace Key` — Derive from `issue.identifier` by replacing any character not in
  `[A-Za-z0-9._-]` with `_`. Use the sanitized value for the workspace directory name.
- `Normalized Issue State` — Compare states after `trim` + `lowercase`.

## 5. Claim Mechanism

### 5.1 Assignment-Based Claiming

T.E.M.P.A.D. uses issue tracker assignment as the distributed claim mechanism. No central server
is needed because the tracker is the shared source of truth.

Claim flow:

1. T.E.M.P.A.D. fetches candidate issues (active state, configured project).
2. An issue is available only if it has no assignee (or the assignee is the current user for
   resumption).
3. When a developer picks a task (TUI) or the orchestrator selects one (daemon), T.E.M.P.A.D.
   assigns the issue to the current user's tracker identity.
4. If the assignment fails (someone else claimed it concurrently), skip the issue and move on.

### 5.2 Race Condition Handling

Two T.E.M.P.A.D. instances may attempt to claim the same issue simultaneously:

- After assigning, re-fetch the issue and verify the assignee matches the current user.
- If the assignee is someone else, release the claim (unassign) and skip the issue.
- This is an optimistic concurrency pattern — conflicts are rare and handled gracefully.

### 5.3 Claim Release

A claim is released (issue unassigned) when:

- The developer explicitly releases the task (TUI mode).
- Daemon mode determines the issue is no longer eligible (terminal state, non-active state).
- Daemon mode exhausts retries without success.

Note: In TUI mode, after opening the IDE, T.E.M.P.A.D. does not automatically release the claim.
The developer (or their agent) is responsible for managing the issue state from that point.

## 6. Workflow Specification (Repository Contract)

### 6.1 File Discovery and Path Resolution

Workflow file path precedence:

1. Explicit CLI argument (`tempad --workflow /path/to/WORKFLOW.md`).
2. Default: `WORKFLOW.md` in the current process working directory.

Loader behavior:

- If the file cannot be read, return `missing_workflow_file` error.
- The workflow file is expected to be repository-owned and version-controlled.

### 6.2 File Format

`WORKFLOW.md` is a Markdown file with optional YAML front matter.

Design note:

- `WORKFLOW.md` should be self-contained enough to describe the team's workflow (prompt, runtime
  settings, hooks, and tracker selection) without requiring out-of-band configuration beyond user
  preferences.

Parsing rules:

- If file starts with `---`, parse lines until the next `---` as YAML front matter.
- Remaining lines become the prompt body.
- If front matter is absent, treat the entire file as prompt body and use an empty config map.
- YAML front matter must decode to a map/object; non-map YAML is an error.
- Prompt body is trimmed before use.

Returned workflow object:

- `config`: front matter root object (not nested under a `config` key).
- `prompt_template`: trimmed Markdown body.

### 6.3 Front Matter Schema

Top-level keys:

- `tracker`
- `polling`
- `workspace`
- `hooks`
- `agent`

Unknown keys should be ignored for forward compatibility.

#### 6.3.1 `tracker` (object)

Fields:

- `kind` (string)
  - Required for dispatch.
  - Current supported value: `linear`
  - Designed for future extensibility (Jira, Asana, GitHub Issues, etc.).
- `endpoint` (string)
  - Default for `tracker.kind == "linear"`: `https://api.linear.app/graphql`
- `api_key` (string)
  - May be a literal token or `$VAR_NAME`.
  - Canonical environment variable for `tracker.kind == "linear"`: `LINEAR_API_KEY`.
  - User config (`~/.tempad/config.yaml`) may also provide this.
  - If `$VAR_NAME` resolves to an empty string, treat the key as missing.
- `project_slug` (string)
  - Required for dispatch when `tracker.kind == "linear"`.
- `active_states` (list of strings or comma-separated string)
  - Default: `Todo`, `In Progress`
- `terminal_states` (list of strings or comma-separated string)
  - Default: `Closed`, `Cancelled`, `Canceled`, `Duplicate`, `Done`

#### 6.3.2 `polling` (object)

Fields:

- `interval_ms` (integer or string integer)
  - Default: `30000`
  - Changes should be re-applied at runtime and affect future tick scheduling without restart.

#### 6.3.3 `workspace` (object)

Fields:

- `root` (path string or `$VAR`)
  - Default: `<system-temp>/tempad_workspaces`
  - `~` and strings containing path separators are expanded.

#### 6.3.4 `hooks` (object)

Fields:

- `after_create` (multiline shell script string, optional)
  - Runs only when a workspace directory is newly created.
  - Failure aborts workspace creation.
- `before_run` (multiline shell script string, optional)
  - Runs before each agent attempt (daemon) or before IDE open (TUI).
  - Failure aborts the current attempt / IDE open.
- `after_run` (multiline shell script string, optional)
  - Runs after each agent attempt in daemon mode (success, failure, timeout, or cancellation).
  - Not triggered in TUI mode (T.E.M.P.A.D. does not track IDE session lifecycle).
  - Failure is logged but ignored.
- `before_remove` (multiline shell script string, optional)
  - Runs before workspace deletion if the directory exists.
  - Failure is logged but ignored; cleanup still proceeds.
- `timeout_ms` (integer, optional)
  - Default: `60000`
  - Applies to all workspace hooks.

#### 6.3.5 `agent` (object)

Fields for daemon mode agent lifecycle:

- `command` (string shell command)
  - Default: none (must be configured in `WORKFLOW.md` or `~/.tempad/config.yaml` for daemon mode).
  - The runtime launches this command via `bash -lc <command>` in the workspace directory.
  - Examples: `codex app-server`, `claude-code --auto`, `opencode`.
  - User config `agent.command` overrides this if set.
- `prompt_delivery` (string)
  - How the rendered prompt is delivered to the agent.
  - Values: `stdin`, `file`, `arg`, `env`.
  - Default: `file` (writes prompt to `PROMPT.md` in the workspace before launching).
  - `stdin`: pipes the prompt to the agent's stdin.
  - `file`: writes to `<workspace>/PROMPT.md` and passes the path as the first argument.
  - `arg`: passes the prompt as a CLI argument.
  - `env`: sets the `TEMPAD_PROMPT` environment variable.
- `max_concurrent` (integer or string integer)
  - Default: `5`
  - Daemon mode only. Changes re-applied at runtime.
- `max_concurrent_by_state` (map `state_name -> positive integer`)
  - Default: empty map.
  - State keys are normalized (`trim` + `lowercase`) for lookup.
  - Invalid entries (non-positive or non-numeric) are ignored.
  - Daemon mode only.
- `max_turns` (integer or string integer)
  - Default: `20`
  - Daemon mode only. Maximum agent turns per workspace session.
- `max_retries` (integer or string integer)
  - Default: `10`
  - Maximum failure-driven retry attempts before releasing the claim.
  - Daemon mode only.
- `max_retry_backoff_ms` (integer or string integer)
  - Default: `300000` (5 minutes)
  - Changes re-applied at runtime.
- `turn_timeout_ms` (integer)
  - Default: `3600000` (1 hour)
  - Daemon mode only.
- `stall_timeout_ms` (integer)
  - Default: `300000` (5 minutes)
  - If `<= 0`, stall detection is disabled.
  - Daemon mode only.
- `read_timeout_ms` (integer)
  - Default: `5000`
  - Timeout for structured protocol handshake responses (if the agent uses a protocol).
  - Daemon mode only.

### 6.4 Prompt Template Contract

The Markdown body of `WORKFLOW.md` is the per-issue prompt template.

Rendering requirements:

- Use a strict template engine (Liquid-compatible semantics are sufficient).
- Unknown variables must fail rendering.
- Unknown filters must fail rendering.

Template input variables:

- `issue` (object) — Includes all normalized issue fields.
- `attempt` (integer or null) — `null` on first attempt, integer on retry/continuation.

Fallback prompt behavior:

- If the workflow prompt body is empty, the runtime may use a minimal default prompt.
- Workflow file read/parse failures are configuration errors and should not silently fall back.

### 6.5 Workflow Validation and Error Surface

Error classes:

- `missing_workflow_file`
- `workflow_parse_error`
- `workflow_front_matter_not_a_map`
- `template_parse_error`
- `template_render_error`

Dispatch gating behavior:

- Workflow file read/YAML errors block new dispatches until fixed.
- Template errors fail only the affected run attempt.

## 7. User Configuration

### 7.1 File Location

`~/.tempad/config.yaml`

Created automatically with sensible defaults on first run if absent.

### 7.2 Schema

```yaml
# Tracker identity — who you are in Linear
tracker:
  identity: "user@example.com"  # or Linear user ID
  api_key: "$LINEAR_API_KEY"    # override repo-level key

# IDE preferences
ide:
  command: "code"         # code, cursor, zed, idea, webstorm, etc.
  args: null              # extra args, e.g., "--new-window"

# Default agent for daemon mode
agent:
  command: "claude-code --auto"
  args: null

# Display preferences
display:
  theme: "auto"           # auto, dark, light
```

### 7.3 Resolution

- CLI flags override user config.
- User config overrides repo config for personal preference fields.
- Repo config overrides user config for team-shared settings (hooks, states, workspace root).
- `$VAR` indirection supported in `api_key` fields.

## 8. Configuration Specification

### 8.1 Source Precedence

1. CLI flags (highest priority).
2. User config (`~/.tempad/config.yaml`) for personal preferences.
3. Repo config (`WORKFLOW.md` front matter) for team settings.
4. Environment variable indirection via `$VAR_NAME`.
5. Built-in defaults (lowest priority).

### 8.2 Dynamic Reload Semantics

- The software should watch `WORKFLOW.md` for changes.
- On change, re-read and re-apply workflow config and prompt template without restart.
- Invalid reloads keep last known good configuration and emit an operator-visible error.
- Reloaded config applies to future dispatch, hook execution, and agent launches.
- In-flight agent sessions are not restarted on config change.

### 8.3 Dispatch Preflight Validation

Startup validation:

- Validate configuration before starting.
- If startup validation fails, fail startup with an operator-visible error.

Per-tick validation (daemon mode):

- Re-validate before each dispatch cycle.
- If validation fails, skip dispatch for that tick, keep reconciliation active.

Validation checks:

- Workflow file can be loaded and parsed.
- `tracker.kind` is present and supported.
- `tracker.api_key` is present after `$` resolution.
- `tracker.project_slug` is present when required by the selected tracker kind.
- `tracker.identity` is present (from user config or CLI).
- `agent.command` is present for daemon mode.

### 8.4 Config Fields Summary

Repo-level (`WORKFLOW.md` front matter):

- `tracker.kind`: string, required, currently `linear`
- `tracker.endpoint`: string, default `https://api.linear.app/graphql`
- `tracker.api_key`: string or `$VAR`, canonical env `LINEAR_API_KEY`
- `tracker.project_slug`: string, required when `tracker.kind=linear`
- `tracker.active_states`: list/string, default `Todo, In Progress`
- `tracker.terminal_states`: list/string, default `Closed, Cancelled, Canceled, Duplicate, Done`
- `polling.interval_ms`: integer, default `30000`
- `workspace.root`: path, default `<system-temp>/tempad_workspaces`
- `hooks.after_create`: shell script or null
- `hooks.before_run`: shell script or null
- `hooks.after_run`: shell script or null
- `hooks.before_remove`: shell script or null
- `hooks.timeout_ms`: integer, default `60000`
- `agent.command`: shell command string, no default
- `agent.prompt_delivery`: string, default `file`
- `agent.max_concurrent`: integer, default `5`
- `agent.max_concurrent_by_state`: map of positive integers, default `{}`
- `agent.max_retries`: integer, default `10`
- `agent.max_turns`: integer, default `20`
- `agent.max_retry_backoff_ms`: integer, default `300000`
- `agent.turn_timeout_ms`: integer, default `3600000`
- `agent.stall_timeout_ms`: integer, default `300000`
- `agent.read_timeout_ms`: integer, default `5000`

User-level (`~/.tempad/config.yaml`):

- `tracker.identity`: string, required
- `tracker.api_key`: string or `$VAR` (overrides repo-level)
- `ide.command`: string, default `code`
- `ide.args`: string or null
- `agent.command`: string (overrides repo-level)
- `agent.args`: string or null
- `display.theme`: string, default `auto`

## 9. TUI Mode Specification

### 9.1 Startup

1. Load and validate config (repo + user).
2. Verify tracker credentials.
3. Fetch initial candidate issues.
4. Render the task board.
5. Start the poll loop to refresh the task list.

### 9.2 Task Board Display

The TUI displays available tasks in a ranked list:

- Each task shows: identifier, title, priority indicator, state, labels.
- Tasks are sorted by: priority ascending (1 highest), then oldest `created_at`, then identifier.
- Only unassigned issues in active states are shown as available.
- Issues assigned to the current user are shown separately as "My Active Tasks".
- Issues with non-terminal blockers in `Todo` state are shown but marked as blocked.

### 9.3 Task Selection Flow

1. Developer navigates to a task and selects it.
2. T.E.M.P.A.D. claims the task:
   a. Assign the issue to the current user's tracker identity.
   b. Re-fetch the issue to verify assignment succeeded.
   c. If someone else claimed it, show a message and return to the task board.
3. T.E.M.P.A.D. prepares the workspace:
   a. Create or reuse the per-issue workspace directory.
   b. Run `after_create` hook if the workspace was newly created.
   c. Run `before_run` hook.
4. T.E.M.P.A.D. opens the IDE:
   a. Execute: `<ide.command> <ide.args> <workspace_path>`
   b. The IDE opens with the workspace directory.
5. T.E.M.P.A.D. returns to the task board. The developer and their agent work inside the IDE.

### 9.4 TUI Refresh

- The task list refreshes every `polling.interval_ms`.
- A manual refresh keybinding is available.
- The TUI updates in-place without disrupting the developer's selection state.

### 9.5 TUI Actions

Available keybindings/actions:

- Select/pick a task
- Release a claimed task (unassign)
- Manual refresh
- View task details (description, labels, blockers)
- Open task URL in browser
- Quit

## 10. Daemon Mode Specification

### 10.1 Overview

Daemon mode (`--daemon`) runs T.E.M.P.A.D. without human interaction. It auto-selects tasks and
runs agents headlessly with full lifecycle management.

### 10.2 Orchestration State Machine

The orchestrator is the only component that mutates scheduling state.

#### 10.2.1 Issue Orchestration States (Internal)

1. `Unclaimed` — Issue is not running and has no retry scheduled.
2. `Claimed` — Orchestrator has reserved the issue (assigned in tracker).
3. `Running` — Agent subprocess exists and issue is tracked in `running` map.
4. `RetryQueued` — Agent is not running, but a retry timer exists.
5. `Released` — Claim removed because issue is terminal, non-active, or retry exhausted.

#### 10.2.2 Run Attempt Lifecycle

1. `PreparingWorkspace`
2. `BuildingPrompt`
3. `LaunchingAgent`
4. `AgentRunning`
5. `Finishing`
6. `Succeeded`
7. `Failed`
8. `TimedOut`
9. `Stalled`
10. `CanceledByReconciliation`

### 10.3 Poll Loop

At startup, the service validates config, performs startup cleanup, schedules an immediate tick, and
repeats every `polling.interval_ms`.

Tick sequence:

1. Reconcile running issues.
2. Run dispatch preflight validation.
3. Fetch candidate issues from tracker (unassigned, active-state).
4. Sort issues by dispatch priority.
5. Dispatch eligible issues while concurrency slots remain.
6. Update observability/status.

### 10.4 Candidate Selection Rules

An issue is dispatch-eligible only if all are true:

- It has `id`, `identifier`, `title`, and `state`.
- Its state is in `active_states` and not in `terminal_states`.
- It is unassigned, or assigned to the current user (for resumption).
- It is not already in `running` or `claimed`.
- Global concurrency slots are available.
- Blocker rule for `Todo` state passes:
  - If the issue state is `Todo`, do not dispatch when any blocker is non-terminal.

Sorting order:

1. `priority` ascending (1..4 preferred; null sorts last)
2. `created_at` oldest first
3. `identifier` lexicographic tie-breaker

### 10.5 Concurrency Control

Global limit:

- `available_slots = max(agent.max_concurrent - running_count, 0)`

Per-state limit (optional):

- `agent.max_concurrent_by_state[state]` if present (state key normalized via `trim` + `lowercase`).
- Otherwise fallback to global limit.
- Invalid entries (non-positive or non-numeric) are ignored.
- The runtime counts issues by their current tracked state in the `running` map.

### 10.6 Retry and Backoff

Retry entry creation:

- Cancel any existing retry timer for the same issue.
- Store `attempt`, `identifier`, `error`, `due_at_ms`, and timer handle.

Backoff formula:

- Normal continuation retries after a clean exit: `1000` ms fixed delay.
- Failure-driven retries: `delay = min(10000 * 2^(attempt - 1), agent.max_retry_backoff_ms)`.
- Power is capped by the configured max retry backoff (default `300000` / 5 minutes).

Maximum retry attempts:

- `agent.max_retries` (default: `10`).
- After exhausting max retries, the claim is released (issue unassigned) and the issue returns to
  the available pool.
- Continuation retries (after normal exit) do not count toward the retry limit — only
  failure-driven retries are counted.

Retry handling:

1. Fetch active candidate issues.
2. Find the specific issue by `issue_id`.
3. If not found or no longer eligible, release claim (unassign).
4. If found and eligible: dispatch if slots available, otherwise requeue.
5. If retry attempt exceeds `agent.max_retries`, release claim and log the exhaustion.

### 10.7 Active Run Reconciliation

Runs every tick with two parts.

Part A — Stall detection:

- For each running issue, check elapsed time since last agent output.
- If elapsed > `agent.stall_timeout_ms`, terminate the agent and queue retry.
- If `stall_timeout_ms <= 0`, skip stall detection.

Part B — Tracker state refresh:

- Fetch current issue states for all running issue IDs.
- Terminal state: terminate agent and clean workspace.
- Still active: update in-memory snapshot.
- Neither active nor terminal: terminate agent without workspace cleanup.
- If state refresh fails, keep agents running and retry next tick.

### 10.8 Startup Terminal Workspace Cleanup

When the service starts:

1. Query tracker for issues in terminal states.
2. Remove corresponding workspace directories.
3. If the fetch fails, log a warning and continue.

## 11. Agent Launcher

### 11.1 Agent-Agnostic Design

T.E.M.P.A.D. does not mandate a specific coding agent. The agent is configured as a shell command
that runs in a workspace directory.

### 11.2 TUI Mode Launch

In TUI mode, the "agent" is the developer's IDE. T.E.M.P.A.D. opens it:

```text
bash -lc "<ide.command> <ide.args> <workspace_path>"
```

The developer and their preferred agent (running inside the IDE) handle everything from there.

### 11.3 Daemon Mode Launch

In daemon mode, T.E.M.P.A.D. launches the agent as a subprocess:

```text
bash -lc "<agent.command> <agent.args>"
```

Working directory: the per-issue workspace path.

Prompt delivery depends on `agent.prompt_delivery`:

- `file`: Write the rendered prompt to `<workspace>/PROMPT.md` before launch.
- `stdin`: Pipe the rendered prompt to the agent's stdin.
- `arg`: Pass the rendered prompt as a CLI argument.
- `env`: Set the `TEMPAD_PROMPT` environment variable.

### 11.4 Agent Output Handling (Daemon Mode)

The orchestrator monitors the agent subprocess:

- Stdout and stderr are captured for logging/observability.
- Agent exit code determines success (0) or failure (non-zero).
- This is the minimum contract: launch, wait for exit, check exit code.

Optional structured output (JSON lines on stdout):

- If the agent emits JSON lines on stdout, implementations may parse them for richer observability.
- Recommended structured event fields (if emitted):
  - `event` (string) — event type (e.g., `turn_completed`, `notification`, `error`)
  - `timestamp` (ISO-8601) — when the event occurred
  - `message` (string) — human-readable summary
  - `usage` (object, optional) — `{input_tokens, output_tokens, total_tokens}`
- Stderr is not part of the structured stream — log it as diagnostics only.
- Non-JSON stdout lines should be treated as plain log output, not protocol violations.
- Implementations must tolerate agents that produce no structured output at all.

Multi-turn agents:

- Some agents (e.g., Codex app-server) support multi-turn sessions via a protocol over stdio.
- T.E.M.P.A.D. does not mandate multi-turn support — single-subprocess-per-attempt is conforming.
- If an implementation chooses to support a specific agent protocol (e.g., Codex app-server
  JSON-RPC), it should document the protocol integration as an extension.
- Agents that require multi-turn interaction should run in auto/batch mode where they manage their
  own turn loop internally.

### 11.5 Agent Environment

The agent subprocess inherits the parent environment plus:

- `TEMPAD_ISSUE_ID` — tracker-internal issue ID
- `TEMPAD_ISSUE_IDENTIFIER` — human-readable issue key (e.g., `ABC-123`)
- `TEMPAD_ISSUE_TITLE` — issue title
- `TEMPAD_ISSUE_URL` — issue URL (if available)
- `TEMPAD_WORKSPACE` — absolute workspace path
- `TEMPAD_ATTEMPT` — attempt number (null/0 for first, 1+ for retries)
- `TEMPAD_PROMPT_FILE` — path to the rendered prompt file (if `prompt_delivery=file`)

## 12. Workspace Management and Safety

### 12.1 Workspace Layout

- Workspace root: `workspace.root` (from config).
- Per-issue path: `<workspace_root>/<sanitized_issue_identifier>`
- Workspaces are reused across runs for the same issue.
- Successful runs do not auto-delete workspaces.

### 12.2 Workspace Creation and Reuse

1. Sanitize identifier to `workspace_key`.
2. Compute workspace path under workspace root.
3. Ensure the path exists as a directory.
4. If newly created (`created_now=true`), run `after_create` hook if configured.
5. If an existing non-directory path occupies the workspace location, handle safely (replace or
   fail per implementation policy).

Notes:

- This section does not assume any specific repository/VCS workflow.
- Workspace preparation beyond directory creation (e.g., `git clone`, dependency install, code
  generation) is implementation-defined and typically handled via hooks.

### 12.3 Optional Workspace Population (Implementation-Defined)

The spec does not require built-in VCS or repository bootstrap behavior.

Implementations may populate or synchronize the workspace using hooks (`after_create` and/or
`before_run`).

Failure handling:

- Workspace population failures return an error for the current attempt.
- If failure happens while creating a brand-new workspace, implementations may remove the partially
  prepared directory.
- Reused workspaces should not be destructively reset on population failure unless that policy is
  explicitly chosen and documented.

### 12.4 Workspace Hooks

Execution contract:

- Execute via `bash -lc <script>` with the workspace directory as `cwd`.
- Hook timeout: `hooks.timeout_ms` (default 60000 ms).
- Log hook start, failures, and timeouts.

Failure semantics:

- `after_create` failure: fatal to workspace creation.
- `before_run` failure: fatal to the current attempt / IDE open.
- `after_run` failure: logged and ignored.
- `before_remove` failure: logged and ignored.

### 12.5 Workspace Persistence

- Workspaces persist after successful runs for debugging and resumption.
- Failed workspaces also persist — they may contain partial work useful for diagnostics.
- Terminal workspace cleanup (Section 10.8) removes workspaces only when the issue reaches a
  terminal state in the tracker.
- Manual cleanup: implementations should provide a way to clean specific or all workspaces
  (e.g., `tempad clean` or `tempad clean <identifier>`).

### 12.6 Safety Invariants

Invariant 1: Agent runs only in the per-issue workspace path.

- Before launching the agent or IDE, validate `cwd == workspace_path`.

Invariant 2: Workspace path must stay inside workspace root.

- Normalize both paths to absolute.
- Require `workspace_path` to have `workspace_root` as a prefix.

Invariant 3: Workspace key is sanitized.

- Only `[A-Za-z0-9._-]` allowed.
- Replace all other characters with `_`.

## 13. Issue Tracker Integration Contract

### 13.1 Required Operations

1. `fetch_candidate_issues()` — Return unassigned issues in active states for the configured
   project.
2. `fetch_issues_by_states(state_names)` — Used for startup terminal cleanup.
3. `fetch_issue_states_by_ids(issue_ids)` — Used for active-run reconciliation (daemon mode).
4. `assign_issue(issue_id, user_identity)` — Claim an issue by assigning it.
5. `unassign_issue(issue_id)` — Release a claim.
6. `fetch_issue(issue_id)` — Fetch a single issue for verification after claiming.

### 13.2 Linear-Specific Semantics

- GraphQL endpoint (default `https://api.linear.app/graphql`).
- Auth token in `Authorization` header.
- `tracker.project_slug` maps to Linear project `slugId`.
- Candidate query filters by project, active states, and unassigned (or assigned to current user).
- Assignment via `issueUpdate` mutation with `assigneeId`.
- Pagination required for candidate issues; page size default: `50`.
- Network timeout: `30000 ms`.

### 13.3 Normalization Rules

- `labels` → lowercase strings.
- `blocked_by` → derived from inverse relations where relation type is `blocks`.
- `priority` → integer only (non-integers become null).
- `assignee` → user ID or email, normalized.
- `created_at` and `updated_at` → ISO-8601 timestamps.

### 13.4 Error Handling

Recommended error categories:

- `unsupported_tracker_kind`
- `missing_tracker_api_key`
- `missing_tracker_project_slug`
- `missing_tracker_identity`
- `tracker_api_request` (transport failures)
- `tracker_api_status` (non-200 HTTP)
- `tracker_api_errors` (GraphQL errors)
- `tracker_claim_conflict` (assignment race lost)

Orchestrator behavior on tracker errors:

- Candidate fetch failure: log and skip dispatch for this tick.
- Claim failure: skip this issue, move to next candidate.
- Reconciliation failure: keep agents running, retry next tick.
- Startup cleanup failure: log warning and continue.

### 13.5 Tracker Writes (Important Boundary)

T.E.M.P.A.D. performs exactly two tracker writes:

1. Assign issue to the current user (claim).
2. Unassign issue (release claim).

All other ticket mutations (state transitions, comments, PR metadata) are handled by the coding
agent using tools defined by the workflow prompt. T.E.M.P.A.D. remains a scheduler/runner and
tracker reader.

### 13.6 Future Adapter Extensibility

The tracker client is designed as an adapter interface:

- `fetch_candidate_issues()`, `assign_issue()`, `unassign_issue()`, etc. are the abstract
  operations.
- `LinearTrackerClient` is the first concrete implementation.
- Future adapters (Jira, GitHub Issues, Asana) implement the same interface with different API
  calls.
- The normalized `Issue` model (Section 4.1.1) is the stable contract between the tracker adapter
  and the rest of the system.

## 14. Prompt Construction

### 14.1 Inputs

- `workflow.prompt_template`
- Normalized `issue` object
- Optional `attempt` integer

### 14.2 Rendering Rules

- Render with strict variable checking.
- Render with strict filter checking.
- Convert issue object keys to strings for template compatibility.
- Preserve nested arrays/maps (labels, blockers) for template iteration.

### 14.3 Retry/Continuation Semantics

`attempt` is passed to the template so the workflow prompt can provide different instructions for
first runs, continuations, and retries.

### 14.4 Failure Semantics

Prompt rendering failure fails the run attempt immediately.

## 15. Logging and Observability

### 15.1 Logging Conventions

Required context fields:

- `issue_id`
- `issue_identifier`
- `mode` (tui or daemon)

For daemon mode agent sessions:

- `attempt`
- `agent_pid` (if available)

### 15.2 Logging Outputs and Sinks

- Operators must see startup/validation/dispatch failures without a debugger.
- Implementations may write to one or more sinks.
- Log sink failures do not crash the service; emit a warning through any remaining sink.

Recommended sink configuration:

- Default: stderr for TUI mode (so it doesn't interfere with TUI rendering), file for daemon mode.
- Log file location: `~/.tempad/logs/tempad.log` (or configurable).
- Agent output logs: `~/.tempad/logs/<issue_identifier>/agent.log` per issue.
- Rotation: implementations should avoid unbounded log growth (size-based rotation recommended).
- Log level: configurable via CLI (`--log-level`) or user config. Default: `info`.

Message formatting:

- Use stable `key=value` phrasing.
- Include action outcome (`completed`, `failed`, `retrying`, etc.).
- Include concise failure reason when present.
- Avoid logging large raw payloads unless at `debug` level.

### 15.3 TUI Status (TUI Mode)

The TUI itself is the primary observability surface:

- Available tasks, claimed tasks, workspace status.
- Error messages for failed claims, hook failures, etc.

### 15.4 Session Metrics and Token Accounting (Daemon Mode)

If the agent provides structured output with token counts:

- Prefer absolute thread totals when available.
- Track deltas relative to last reported totals to avoid double-counting.
- Accumulate aggregate totals in orchestrator state.
- Do not treat generic `usage` maps as cumulative unless the event type defines them that way.

Runtime accounting:

- Report runtime as a live aggregate at snapshot/render time.
- Add run duration seconds to cumulative totals when a session ends.

### 15.5 Optional HTTP Server Extension

If implemented:

- Enabled via CLI `--port` or config `server.port`.
- `server.port` is extension configuration, not part of the core front-matter schema.
- Precedence: CLI `--port` overrides `server.port`.
- Bind loopback by default (`127.0.0.1`) unless explicitly configured otherwise.
- `server.port = 0` may be used for ephemeral port binding in tests.
- Changes to port do not need hot-rebind; restart-required behavior is conformant.
- Must not be required for correctness.

If the HTTP server is implemented, provide these minimum endpoints:

- `GET /` — Human-readable dashboard (server-rendered or client-side app).
- `GET /api/v1/state` — JSON summary of running sessions, retry queue, aggregate totals.
- `GET /api/v1/<issue_identifier>` — Issue-specific runtime/debug details. Return `404` if unknown.
- `POST /api/v1/refresh` — Queue an immediate poll + reconciliation cycle (`202 Accepted`).

API design notes:

- Endpoints are read-only except for operational triggers like `/refresh`.
- Unsupported methods return `405 Method Not Allowed`.
- Errors use `{"error": {"code": "...", "message": "..."}}` envelope.

## 16. Failure Model and Recovery

### 16.1 Failure Classes

1. Workflow/Config Failures — missing file, invalid YAML, missing credentials.
2. Workspace Failures — directory creation, hook failures, path violations.
3. Agent Failures (daemon mode) — launch failure, non-zero exit, timeout, stall.
4. Tracker Failures — API errors, claim conflicts, malformed payloads.
5. Observability Failures — TUI render errors, log sink failures.

### 16.2 Recovery Behavior

- Config failures: show error in TUI / skip dispatch in daemon mode. Keep alive.
- Agent failures (daemon): retry with exponential backoff.
- Tracker fetch failures: skip this tick, try next.
- Claim conflicts: skip issue, pick next candidate.
- TUI/log failures: do not crash.

### 16.3 Restart Recovery

In-memory state is intentionally non-persistent.

After restart:

- No retry timers are restored.
- No running sessions are assumed recoverable.
- Service recovers by: startup terminal workspace cleanup, fresh polling, re-dispatching.

### 16.4 Operator Intervention

- Edit `WORKFLOW.md` → auto-detected and re-applied.
- Change issue states in tracker → agents stopped on reconciliation (daemon mode).
- Restart the service for process recovery.
- In TUI mode: release tasks manually, refresh the board.

## 17. Security and Operational Safety

### 17.1 Trust Boundary

Each deployment defines its own trust boundary. T.E.M.P.A.D. runs on the developer's own machine
with their own credentials.

### 17.2 Filesystem Safety

- Workspace path must remain under workspace root.
- Agent cwd must be the per-issue workspace path.
- Workspace directory names use sanitized identifiers.

### 17.3 Secret Handling

- Support `$VAR` indirection in config.
- Do not log API tokens or secrets.
- Validate secret presence without printing values.

### 17.4 Hook Script Safety

- Hooks are trusted configuration from `WORKFLOW.md`.
- Hooks run inside the workspace directory.
- Hook output should be truncated in logs.
- Hook timeouts prevent hanging.

### 17.5 Credential Isolation

- T.E.M.P.A.D. passes the tracker API key to the agent subprocess via environment variable
  inheritance. The agent process inherits the full parent environment.
- Implementations should document which credentials the agent has access to.
- For sensitive environments, consider running agents under a restricted OS user or within a
  container to limit credential exposure.
- T.E.M.P.A.D. itself does not sandbox the agent — sandboxing is the agent's responsibility or the
  host environment's.

### 17.6 Harness Hardening Guidance

Running coding agents against repositories, issue trackers, and other inputs that may contain
sensitive data or externally-controlled content can be dangerous. A permissive deployment can lead
to data leaks, destructive mutations, or full machine compromise if the agent is induced to execute
harmful commands.

Implementations should explicitly evaluate their own risk profile. This specification does not
mandate a single hardening posture, but deployments should not assume that tracker data, repository
contents, prompt inputs, or tool arguments are fully trustworthy.

Possible hardening measures include:

- Tightening agent approval and sandbox settings instead of running fully autonomous.
- Adding external isolation layers (OS/container/VM sandboxing, network restrictions).
- Filtering which issues, projects, or labels are eligible for dispatch.
- Reducing the set of credentials, filesystem paths, and network destinations available to the
  agent.
- Using `before_run` hooks to enforce additional workspace constraints.

## 18. CLI Specification

### 18.1 Commands

```text
tempad                        # Start in TUI mode (default)
tempad --daemon               # Start in daemon mode
tempad --workflow <path>      # Use a specific WORKFLOW.md
tempad --port <port>          # Enable HTTP server extension
tempad --identity <identity>  # Override tracker identity
tempad --agent <command>      # Override agent command (daemon mode)
tempad --ide <command>        # Override IDE command (TUI mode)
tempad init                   # Create ~/.tempad/config.yaml with defaults
tempad validate               # Validate config without starting
tempad clean                  # Remove all workspaces for terminal issues
tempad clean <identifier>     # Remove workspace for a specific issue
```

### 18.2 Exit Behavior

- TUI mode: exits when the user quits.
- Daemon mode: runs until interrupted (SIGINT/SIGTERM).
- Exit 0 on normal shutdown.
- Exit non-zero on startup failure.

## 19. Reference Algorithms

### 19.1 Service Startup

```text
function start_service(mode):
  configure_logging()
  load_user_config("~/.tempad/config.yaml")
  load_workflow("WORKFLOW.md")
  start_workflow_watch(on_change=reload_and_reapply)

  validation = validate_config(mode)
  if validation is not ok:
    fail_startup(validation)

  startup_terminal_workspace_cleanup()

  if mode == "tui":
    start_tui()
  else if mode == "daemon":
    start_daemon()
```

### 19.2 TUI Task Selection

```text
function on_task_selected(issue):
  claim_result = tracker.assign_issue(issue.id, config.tracker.identity)
  if claim_result failed:
    show_error("Failed to claim task")
    return

  verification = tracker.fetch_issue(issue.id)
  if verification.assignee != config.tracker.identity:
    tracker.unassign_issue(issue.id)
    show_error("Someone else claimed this task")
    return

  workspace = workspace_manager.create_for_issue(issue.identifier)
  if workspace failed:
    show_error("Workspace creation failed")
    return

  if run_hook("before_run", workspace.path) failed:
    show_error("before_run hook failed")
    return

  launch_ide(config.ide.command, config.ide.args, workspace.path)
  show_message("Opened in IDE. You're in charge now.")
```

### 19.3 Daemon Poll-and-Dispatch

```text
on_tick(state):
  state = reconcile_running_issues(state)

  validation = validate_dispatch_config()
  if validation is not ok:
    log_error(validation)
    schedule_tick(state.poll_interval_ms)
    return state

  issues = tracker.fetch_candidate_issues()
  if issues failed:
    log_error("tracker fetch failed")
    schedule_tick(state.poll_interval_ms)
    return state

  for issue in sort_for_dispatch(issues):
    if no_available_slots(state):
      break
    if should_dispatch(issue, state):
      state = claim_and_dispatch(issue, state)

  schedule_tick(state.poll_interval_ms)
  return state
```

### 19.4 Daemon Claim-and-Dispatch

```text
function claim_and_dispatch(issue, state):
  claim = tracker.assign_issue(issue.id, config.tracker.identity)
  if claim failed:
    return state  # skip, someone else got it

  verification = tracker.fetch_issue(issue.id)
  if verification.assignee != config.tracker.identity:
    return state  # race lost, move on

  worker = spawn_agent_worker(issue, attempt=null)
  if worker failed:
    tracker.unassign_issue(issue.id)
    return schedule_retry(state, issue, error="spawn failed")

  state.running[issue.id] = new_running_entry(issue, worker)
  state.claimed.add(issue.id)
  return state
```

### 19.5 Agent Worker (Daemon Mode)

```text
function run_agent_worker(issue, attempt):
  workspace = workspace_manager.create_for_issue(issue.identifier)
  if workspace failed:
    fail_worker("workspace error")

  if run_hook("before_run", workspace.path) failed:
    fail_worker("before_run hook error")

  prompt = render_prompt(workflow.prompt_template, issue, attempt)
  if prompt failed:
    fail_worker("prompt error")

  deliver_prompt(prompt, workspace.path, config.agent.prompt_delivery)

  agent_process = launch_agent(config.agent.command, workspace.path)
  if agent_process failed:
    run_hook_best_effort("after_run", workspace.path)
    fail_worker("agent launch error")

  result = wait_for_agent(agent_process, timeout=config.agent.turn_timeout_ms)

  run_hook_best_effort("after_run", workspace.path)

  if result.exit_code == 0:
    exit_normal()
  else:
    fail_worker("agent exited with code " + result.exit_code)
```

### 19.6 Worker Exit and Retry

```text
on_worker_exit(issue_id, reason, state):
  running_entry = state.running.remove(issue_id)

  if reason == normal:
    state.completed.add(issue_id)
    # Schedule continuation check — is the issue still active?
    state = schedule_retry(state, issue_id, 1, {
      delay_type: continuation,
      delay_ms: 1000
    })
  else:
    state = schedule_retry(state, issue_id, next_attempt(running_entry), {
      error: reason
    })

  return state

on_retry_timer(issue_id, state):
  retry_entry = state.retry_attempts.pop(issue_id)
  if missing:
    return state

  issue = tracker.fetch_issue(issue_id)
  if issue is null or issue.state not in active_states:
    tracker.unassign_issue(issue_id)
    state.claimed.remove(issue_id)
    return state

  if available_slots(state) == 0:
    return schedule_retry(state, issue_id, retry_entry.attempt + 1, {
      error: "no slots"
    })

  return dispatch_issue(issue, state, retry_entry.attempt)
```

## 20. Test and Validation Matrix

### 20.1 Core Conformance

**Config and Workflow:**

- Workflow file path precedence (CLI > cwd default).
- Dynamic reload on `WORKFLOW.md` change.
- Invalid reload keeps last known good config.
- User config merges with repo config correctly.
- `$VAR` resolution for API keys and paths.
- Config defaults apply when values are missing.

**Claim Mechanism:**

- Assignment-based claiming assigns issue to current user.
- Claim verification re-fetches and checks assignee.
- Race condition: concurrent claim detected and handled gracefully.
- Claim release unassigns the issue.

**Workspace:**

- Deterministic workspace path per issue identifier.
- New workspace triggers `after_create` hook.
- Existing workspace is reused without `after_create`.
- `before_run` failure aborts the attempt.
- Path sanitization enforced.
- Root containment invariant enforced.

**TUI Mode:**

- Available tasks displayed sorted by priority.
- Only unassigned active-state issues shown.
- Task selection triggers claim + workspace + IDE open.
- Failed claim shows error and returns to board.
- Blocked issues marked as blocked.

**Daemon Mode:**

- Auto-dispatch respects global concurrency limits.
- Per-state concurrency limits enforced when configured.
- `Todo` with non-terminal blockers not dispatched.
- Normal exit schedules continuation retry (does not count toward max retries).
- Abnormal exit schedules exponential backoff retry.
- Retry backoff cap uses configured `agent.max_retry_backoff_ms`.
- Max retries exhausted releases claim and unassigns issue.
- Stall detection terminates stalled agents and schedules retry.
- Terminal state stops agent and cleans workspace.
- Non-active state stops agent without cleanup.
- Agent environment variables are set correctly (`TEMPAD_ISSUE_ID`, etc.).

**Tracker Integration:**

- Candidate fetch filters by project, active states, unassigned.
- Assignment mutation works.
- Pagination preserves order.
- Error categories mapped correctly.

**CLI:**

- TUI mode is default (no flags).
- `--daemon` enables daemon mode.
- `--workflow <path>` overrides workflow file location.
- `./WORKFLOW.md` is used when no workflow path is provided.
- CLI errors on nonexistent explicit workflow path or missing default.
- `tempad validate` validates config without starting.
- `tempad init` creates user config with defaults.
- `tempad clean` removes workspaces for terminal issues.
- Exit 0 on normal shutdown, non-zero on startup failure.

### 20.2 Extension Conformance

- HTTP server serves state API and dashboard if enabled.
- HTTP server binds loopback by default.
- Agent structured output parsing (if implemented).
- Per-state concurrency limits (if configured).

### 20.3 Real Integration Profile

- Smoke test with real Linear credentials.
- Isolated test identifiers and cleanup.
- Hook execution on target OS/shell.

## 21. Implementation Checklist

### 21.1 Required for Conformance

- CLI with TUI mode (default) and daemon mode (`--daemon`).
- `WORKFLOW.md` loader with YAML front matter + prompt body.
- User config at `~/.tempad/config.yaml`.
- Merged config layer with correct precedence.
- Dynamic `WORKFLOW.md` watch/reload.
- Issue tracker client with candidate fetch + assignment + state refresh.
- Assignment-based claim mechanism with race detection.
- Workspace manager with sanitized per-issue workspaces.
- Workspace lifecycle hooks.
- Agent-agnostic launcher (IDE open for TUI, subprocess for daemon).
- Prompt rendering with `issue` and `attempt`.
- Daemon mode orchestrator: polling, dispatch, reconciliation, retries.
- Structured logging with issue context.
- TUI: task board display, selection, claim, IDE open.

### 21.2 Recommended Extensions

- Optional HTTP server for observability (Section 15.5).
- Agent structured output parsing for richer daemon mode observability.
- Agent protocol extensions (e.g., Codex app-server JSON-RPC integration).
- Multiple tracker adapters (Jira, GitHub Issues, Asana).
- Persistent retry queue across restarts.
- TUI themes and customization.
- `tempad clean` command for workspace management.
- Desktop notifications for TUI mode events (new high-priority tasks, agent completion).

---

## Appendix A: Example `WORKFLOW.md`

```markdown
---
tracker:
  kind: linear
  api_key: $LINEAR_API_KEY
  project_slug: my-project

polling:
  interval_ms: 30000

workspace:
  root: ~/workspaces/tempad

hooks:
  after_create: |
    git clone git@github.com:myorg/myrepo.git .
    npm install
  before_run: |
    git fetch origin
    git checkout -b tempad/{{ issue.identifier | downcase }} origin/main
  after_run: |
    echo "Run completed for {{ issue.identifier }}"

agent:
  command: "claude-code --auto"
  prompt_delivery: file
  max_concurrent: 3
  max_turns: 20
  max_retries: 10
  turn_timeout_ms: 3600000
  stall_timeout_ms: 300000
---

You are working on issue **{{ issue.identifier }}: {{ issue.title }}**.

## Task Description

{{ issue.description }}

## Priority

{{ issue.priority | default: "No priority set" }}

## Labels

{% for label in issue.labels %}
- {{ label }}
{% endfor %}

{% if issue.blocked_by.size > 0 %}
## Blockers
{% for blocker in issue.blocked_by %}
- {{ blocker.identifier }} ({{ blocker.state }})
{% endfor %}
{% endif %}

{% if attempt %}
## Retry Context
This is retry attempt #{{ attempt }}. Review previous work in the workspace and continue
from where you left off. Do not start over.
{% endif %}

## Instructions

1. Read the issue description carefully.
2. Implement the required changes in the workspace.
3. Write tests for your changes.
4. Commit your work with a descriptive message referencing {{ issue.identifier }}.
5. Push your branch and create a pull request.
6. Move the issue to "Human Review" state when the PR is ready.
```

## Appendix B: Example `~/.tempad/config.yaml`

```yaml
# T.E.M.P.A.D. User Configuration
# This file stores personal preferences. Team settings belong in WORKFLOW.md.

# Tracker identity — who you are in Linear
tracker:
  identity: "user@example.com"    # Your Linear email or user ID
  api_key: "$LINEAR_API_KEY"      # Override repo-level key if needed

# IDE preferences (TUI mode)
ide:
  command: "cursor"               # code, cursor, zed, idea, webstorm, etc.
  args: "--new-window"            # Extra arguments

# Default agent (daemon mode)
agent:
  command: "claude-code --auto"   # Agent command for headless execution
  args: null                      # Extra arguments

# Display preferences
display:
  theme: "auto"                   # auto, dark, light

# Logging (optional overrides)
logging:
  level: "info"                   # debug, info, warn, error
  file: "~/.tempad/logs/tempad.log"
```

## Appendix C: Comparison with Centralized Architectures

T.E.M.P.A.D. intentionally avoids a central server. This appendix documents the trade-offs.

**What a central server would add:**

- Centralized fleet visibility (one dashboard for all workers).
- Server-side routing strategies (least-loaded, skill-match).
- Single point of coordination for duplicate prevention.

**Why T.E.M.P.A.D. does not need one:**

- The issue tracker (Linear) is already a shared coordination layer. Assignment-based claiming
  provides distributed locking without a custom server.
- Fleet visibility is available in the tracker itself — you can see who is assigned to what.
- Skill matching is better left to developers who know their own strengths.
- No single point of failure — each developer's instance is independent.
- Dramatically simpler architecture — one binary, no network protocol, no deployment coordination.

**Future evolution path:**

- If fleet-wide observability becomes necessary, add a lightweight aggregator that reads from each
  instance's optional HTTP API — not a dispatcher.
- If multi-tracker support is needed, add a thin proxy/adapter layer — not a central server.
- The tracker adapter interface (Section 13.6) is designed for this extensibility.
