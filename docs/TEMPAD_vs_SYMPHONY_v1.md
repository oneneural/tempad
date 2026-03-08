# T.E.M.P.A.D. vs Symphony — Feature Comparison

| | |
|---|---|
| **Version** | 1.0.0 |
| **Date** | 2026-03-08 |

T.E.M.P.A.D. is a superset of Symphony — every Symphony capability is present, plus significant additions.

---

## Quick Summary

Symphony is a headless daemon that polls Linear and runs Codex. T.E.M.P.A.D. takes that same
daemon engine, makes it agent-agnostic, adds an interactive TUI mode, introduces assignment-based
distributed claiming, splits config into repo and user layers, and hardens the lifecycle with retry
limits and richer observability.

---

## Feature Matrix

| Capability | Symphony v1 | T.E.M.P.A.D. |
|---|---|---|
| **Operating Modes** | Daemon only (headless) | Daemon + TUI (interactive task board) |
| **Agent Support** | Codex app-server only (hardcoded JSON-RPC protocol) | Any agent — shell command + exit code contract |
| **Prompt Delivery** | Codex protocol (`turn/start` input) | Four methods: `file`, `stdin`, `arg`, `env` |
| **Multi-Turn Protocol** | Required (Codex app-server thread/turn lifecycle) | Optional extension — single-subprocess-per-attempt is conforming |
| **Claim Mechanism** | None — dispatch is local, no duplicate prevention across machines | Assignment-based claiming via tracker with optimistic concurrency |
| **Race Handling** | Not addressed (single instance assumed) | Assign → verify → release on conflict |
| **Claim Release** | Not applicable | On terminal state, retry exhaustion, or manual release (TUI) |
| **Config Model** | Single source: `WORKFLOW.md` front matter | Split: `WORKFLOW.md` (team/repo) + `~/.tempad/config.yaml` (personal) |
| **Config Precedence** | Front matter → `$VAR` → defaults | CLI → user config → repo config → `$VAR` → defaults |
| **IDE Integration** | None | TUI opens developer's configured IDE (`code`, `cursor`, `zed`, etc.) |
| **Concurrency — Global** | `agent.max_concurrent_agents` (default 10) | `agent.max_concurrent` (default 5) — lower default for developer machines |
| **Concurrency — Per-State** | `agent.max_concurrent_agents_by_state` | `agent.max_concurrent_by_state` — same concept, present in both |
| **Max Retries** | Unlimited — retries forever until issue state changes | `agent.max_retries` (default 10) — releases claim after exhaustion |
| **Continuation Retries** | Short fixed delay (1s) after clean exit | Same — and continuation retries don't count toward `max_retries` |
| **Failure Retries** | Exponential backoff: `min(10000 * 2^(attempt-1), max_retry_backoff_ms)` | Same formula, same default cap (5 min) |
| **Stall Detection** | Based on last Codex event timestamp | Based on last agent output timestamp — agent-agnostic |
| **Reconciliation** | Every tick: stall check + tracker state refresh | Same two-part reconciliation |
| **Startup Cleanup** | Remove workspaces for terminal-state issues | Same |
| **Blocker Awareness** | Skip `Todo` issues with non-terminal blockers | Same |
| **Dispatch Sorting** | Priority → oldest → identifier tiebreak | Same |
| **Workspace Layout** | `<workspace_root>/<sanitized_identifier>` | Same |
| **Workspace Hooks** | `after_create`, `before_run`, `after_run`, `before_remove` | Same four hooks |
| **Hook Timeout** | `hooks.timeout_ms` (default 60s) | Same |
| **Workspace Population** | Implementation-defined (hooks) | Same — VCS-agnostic, hook-driven |
| **Dynamic Reload** | Watch `WORKFLOW.md`, re-apply without restart | Same |
| **Dispatch Preflight** | Validate config each tick before dispatch | Same |
| **Tracker Support** | Linear (with adapter pattern described) | Linear (with explicit adapter extensibility section) |
| **Tracker Writes** | Agent's responsibility (not orchestrator) | Same boundary |
| **Agent Environment Vars** | Codex-specific (thread_id, turn_id, etc.) | `TEMPAD_ISSUE_ID`, `TEMPAD_ISSUE_IDENTIFIER`, `TEMPAD_ISSUE_TITLE`, `TEMPAD_ISSUE_URL`, `TEMPAD_WORKSPACE`, `TEMPAD_ATTEMPT`, `TEMPAD_PROMPT_FILE` |
| **Structured Agent Output** | Required (Codex JSON-RPC protocol over stdout) | Optional — JSON lines parsed if present, plain text tolerated |
| **Token Accounting** | Required (from Codex events) | Optional — if agent reports tokens, same accounting rules apply |
| **Approval/Sandbox Policy** | Codex-specific (`approval_policy`, `thread_sandbox`, `turn_sandbox_policy`) | Not applicable — agent manages its own approval/sandbox |
| **Client-Side Tools** | Optional `linear_graphql` tool via Codex protocol | Not specified — agent uses its own tool ecosystem |
| **User Input Handling** | Hard fail on user input request | Not applicable — agents that need input should run in auto/batch mode |
| **HTTP Server Extension** | Optional (`/`, `/api/v1/state`, `/api/v1/<id>`, `/api/v1/refresh`) | Same endpoints, same optional extension |
| **CLI** | Positional workflow path, `--port` | `tempad`, `--daemon`, `--workflow`, `--port`, `--identity`, `--agent`, `--ide`, `init`, `validate`, `clean` |
| **Log Sinks** | Implementation-defined | Recommended defaults: stderr (TUI), file (daemon), per-issue agent logs, rotation guidance |
| **`read_timeout_ms`** | Present (Codex startup handshake timeout, default 5s) | Present in agent config (protocol handshake timeout, default 5s) |
| **Credential Isolation** | Not explicitly addressed | Explicit section on env-var inheritance, restricted user guidance |
| **Harness Hardening** | Explicit guidance section | Same guidance, adapted for agent-agnostic context |
| **Test Matrix** | 8 sections (core + extension + real integration) | Same coverage areas, adapted for TUI + agent-agnostic design |
| **Restart Recovery** | In-memory only — re-polls tracker, re-dispatches | Same — no persistent DB required |
| **Spec Language** | Language-agnostic | Language-agnostic |

---

## What T.E.M.P.A.D. Adds Over Symphony

### 1. Interactive TUI Mode

Symphony is headless-only. T.E.M.P.A.D. adds a full interactive terminal UI where developers see
available tasks, pick one, and T.E.M.P.A.D. claims it and opens their IDE. The developer and their
chosen agent handle the rest inside the IDE. This is the default mode — daemon is opt-in via
`--daemon`.

### 2. Agent-Agnostic Design

Symphony is hardwired to Codex app-server and its JSON-RPC protocol (thread/start, turn/start,
streaming events). T.E.M.P.A.D. treats the agent as a black box: any executable that accepts a
prompt and returns an exit code. Four prompt delivery methods (`file`, `stdin`, `arg`, `env`) cover
virtually every CLI agent (Claude Code, Codex, OpenCode, Aider, custom scripts).

The Codex-specific concepts — `approval_policy`, `thread_sandbox`, `turn_sandbox_policy`,
`turn_timeout_ms`, `read_timeout_ms` on the agent protocol, multi-turn thread management, the
`linear_graphql` client-side tool — are all absent from T.E.M.P.A.D. because they're agent
internals, not orchestrator concerns. If you're running Codex, you configure those in Codex. If
you're running Claude Code, you configure Claude Code's settings. T.E.M.P.A.D. doesn't care.

### 3. Assignment-Based Distributed Claiming

Symphony assumes a single instance — there's no mechanism to prevent two instances from dispatching
the same issue. T.E.M.P.A.D. uses the tracker's assignment field as a distributed lock: assign the
issue to yourself, verify the assignment stuck, release if someone else got it first. This makes
multi-developer teams safe without needing a central server.

### 4. Split Configuration (Repo + User)

Symphony has one config source: `WORKFLOW.md` front matter. T.E.M.P.A.D. splits config into
repo-level (`WORKFLOW.md` — team workflow, hooks, tracker settings) and user-level
(`~/.tempad/config.yaml` — IDE preference, agent command, tracker identity, API keys). Clear
precedence rules ensure teams own the workflow while developers own their personal setup.

### 5. Max Retry Limit

Symphony retries failed agent runs indefinitely — the only escape is the issue changing state in
the tracker. T.E.M.P.A.D. introduces `agent.max_retries` (default 10). After exhausting retries,
T.E.M.P.A.D. unassigns the issue and returns it to the available pool. Continuation retries (agent
exited cleanly, issue still active) don't count toward this limit — only failure-driven retries do.

### 6. Richer CLI

Symphony has a minimal CLI (positional workflow path, `--port`). T.E.M.P.A.D. adds:

- `--daemon` to opt into headless mode (TUI is default)
- `--identity` to override tracker identity
- `--agent` to override agent command
- `--ide` to override IDE command
- `tempad init` to scaffold `~/.tempad/config.yaml`
- `tempad validate` to check config without starting
- `tempad clean` to remove workspaces for terminal issues

### 7. Explicit Agent Environment Variables

Symphony passes context through the Codex protocol (thread/start params, turn/start input).
T.E.M.P.A.D. sets well-named environment variables (`TEMPAD_ISSUE_ID`, `TEMPAD_WORKSPACE`,
`TEMPAD_ATTEMPT`, etc.) that any agent or script can read — no protocol coupling required.

---

## What Symphony Has That T.E.M.P.A.D. Intentionally Drops

### 1. Codex App-Server Protocol Integration

Symphony specifies the full Codex JSON-RPC protocol in detail: `initialize`, `initialized`,
`thread/start`, `turn/start`, streaming turn processing, multi-turn continuation on the same
thread, approval handling, user-input-required detection, and structured event parsing. T.E.M.P.A.D.
replaces all of this with a simple contract: shell command → exit code. Richer agent protocols are
the agent's concern, not the orchestrator's.

### 2. `linear_graphql` Client-Side Tool

Symphony optionally exposes a `linear_graphql` tool to the Codex session so the agent can make
GraphQL calls to Linear using Symphony's configured auth. T.E.M.P.A.D. does not provide client-side
tools — agents use their own tool ecosystems (MCP servers, CLI tools, etc.).

### 3. In-Process Multi-Turn Thread Management

Symphony's worker runs multiple Codex turns within a single worker process on the same live thread,
with the app-server subprocess staying alive across turns. Both specs have `agent.max_turns` (default
20), but the mechanism differs: Symphony manages turn loops over the Codex JSON-RPC protocol with
thread/turn lifecycle. T.E.M.P.A.D. treats `max_turns` as a daemon-mode setting — if the agent
exits cleanly and the issue is still active, a continuation retry dispatches a new subprocess. The
turn tracking is at the orchestrator level rather than inside a persistent agent session.

### 4. Codex-Specific Domain Model Fields

Symphony's `Live Session` entity tracks Codex-specific fields: `session_id` (thread_id-turn_id),
`codex_app_server_pid`, `last_codex_event`, token counters split by last-reported vs absolute,
`turn_count`, and `codex_rate_limits`. T.E.M.P.A.D. has a simpler running entry because it doesn't
mandate a specific agent protocol. If the agent reports structured output, T.E.M.P.A.D. parses it
opportunistically.

---

## Daemon Mode: Direct Comparison

For the headless/daemon use case specifically, here's how the two specs align:

| Daemon Capability | Symphony | T.E.M.P.A.D. | Verdict |
|---|---|---|---|
| Poll loop | ✅ | ✅ | Equivalent |
| Candidate selection | ✅ | ✅ | Equivalent |
| Priority sorting | ✅ | ✅ | Equivalent |
| Blocker gating | ✅ | ✅ | Equivalent |
| Global concurrency | ✅ | ✅ | Equivalent |
| Per-state concurrency | ✅ | ✅ | Equivalent |
| Exponential backoff | ✅ | ✅ | Equivalent |
| Continuation retry | ✅ | ✅ | Equivalent |
| Max retry limit | ❌ | ✅ | T.E.M.P.A.D. adds |
| Stall detection | ✅ | ✅ | Equivalent |
| Tracker reconciliation | ✅ | ✅ | Equivalent |
| Startup cleanup | ✅ | ✅ | Equivalent |
| Workspace isolation | ✅ | ✅ | Equivalent |
| Workspace hooks (4) | ✅ | ✅ | Equivalent |
| Dynamic config reload | ✅ | ✅ | Equivalent |
| Dispatch preflight | ✅ | ✅ | Equivalent |
| Structured logging | ✅ | ✅ | Equivalent |
| HTTP server extension | ✅ | ✅ | Equivalent |
| Restart recovery | ✅ | ✅ | Equivalent |
| Agent-agnostic | ❌ | ✅ | T.E.M.P.A.D. adds |
| Distributed claiming | ❌ | ✅ | T.E.M.P.A.D. adds |
| Split config | ❌ | ✅ | T.E.M.P.A.D. adds |
| Multi-turn protocol | ✅ (Codex JSON-RPC) | ✅ (subprocess restart) | Both support `max_turns`; different mechanism |
| Client-side tools | ✅ | ❌ (by design) | Symphony has, T.E.M.P.A.D. delegates to agent |

**Bottom line:** T.E.M.P.A.D.'s daemon mode is Symphony's full orchestration engine — same poll
loop, same concurrency model, same retry logic, same reconciliation, same workspace management —
with three additions (agent-agnostic, distributed claiming, retry cap) and two intentional removals
(Codex protocol coupling, client-side tools) that push agent-specific concerns to the agent itself.
