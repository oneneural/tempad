# TEMPAD vs Symphony: Gap Analysis & Feasibility Assessment

| | |
|---|---|
| **Version** | 1.0.0 |
| **Date** | 2026-03-08 |

## 1. Coverage Verdict

**TEMPAD covers 100% of Symphony's orchestration logic.** Every daemon-mode capability — poll loop, candidate selection, priority sorting, blocker gating, concurrency control, exponential backoff, continuation retry, stall detection, reconciliation, startup cleanup, workspace hooks, dynamic reload, dispatch preflight, HTTP server extension — is present and equivalent in the TEMPAD spec.

The four things TEMPAD intentionally drops are all Codex-specific coupling, not orchestration features:

1. Codex JSON-RPC protocol (initialize → thread/start → turn/start → streaming)
2. `linear_graphql` client-side tool
3. In-process multi-turn thread management (TEMPAD uses subprocess restart instead)
4. Codex-specific domain fields (thread_id, turn_id, codex_rate_limits, etc.)

These are the right things to drop for an agent-agnostic design.

---

## 2. Gaps Found (Things Symphony's Code Has That TEMPAD Spec Doesn't Cover)

### 2.1 `tracker.assignee` Filtering (MEDIUM priority)

**Symphony's Elixir implementation** has a `tracker.assignee` config field and `assigned_to_worker` logic that filters candidates by assignee. This lets a Symphony instance only pick up issues assigned to a specific user (or "me"), which is critical for multi-instance deployments even without TEMPAD's active claiming.

**TEMPAD's spec** has `tracker.identity` for claiming, but the candidate fetch query in Section 10.4 says "unassigned, or assigned to the current user (for resumption)." This is close but subtly different — it doesn't support the Symphony pattern of "only dispatch issues assigned to me" as a routing filter.

**Recommendation:** TEMPAD already handles this through claiming (assign → verify → dispatch), but should explicitly document that candidate fetch also filters by `tracker.identity` for the "assigned to current user" resumption case. The spec already implies this; just tighten the language.

### 2.2 `create_comment` and `update_issue_state` Tracker Operations

**Symphony's Elixir tracker behaviour** defines `create_comment/2` and `update_issue_state/2` callbacks that aren't in the Symphony spec but exist in the code. These are used by the `linear_graphql` dynamic tool and workflow hooks.

**TEMPAD** correctly excludes these — its spec says "T.E.M.P.A.D. performs exactly two tracker writes: assign and unassign." The agent handles everything else. No gap here, just noting the divergence from Symphony's implementation.

### 2.3 `agent.max_turns` Mechanism Difference

**Symphony:** Runs multiple Codex turns within a single worker process on the same live thread. The subprocess stays alive across turns. `max_turns` is enforced in-process.

**TEMPAD:** Each turn is a separate subprocess. After clean exit, orchestrator schedules continuation retry (1s delay), re-dispatches if issue still active. `max_turns` limits total subprocess invocations.

**Feasibility concern:** None. TEMPAD's approach is simpler and agent-agnostic. The only cost is subprocess startup overhead per turn (~1-2s for most agents), which is negligible compared to turn execution time.

### 2.4 Symphony's `codex_rate_limits` Tracking

**Symphony** tracks rate limit data from Codex events and exposes it via the HTTP API.

**TEMPAD** doesn't have this because it's Codex-specific. The spec's optional structured output parsing covers generic token accounting but not rate limits. This is fine — rate limiting is an agent concern.

---

## 3. Things TEMPAD Adds That Need Feasibility Validation

### 3.1 TUI Mode — FEASIBLE

**What it needs:**
- Terminal UI library (e.g., Ratatui/Rust, Bubble Tea/Go, tui-rs, Ink/Node, Ratatouille/Elixir)
- Live task board with polling refresh
- Keyboard navigation and selection
- Task detail view
- Concurrent with workspace/IDE operations

**Feasibility:** Straightforward. TUI libraries are mature in every major language. The TUI is a read-only view of tracker data + user selection → claim → workspace → IDE open. No complex state management beyond what the orchestrator already has.

**Risk:** Low. TUI is decoupled from orchestration logic — it's just a presentation layer over the same tracker client and workspace manager.

### 3.2 Assignment-Based Distributed Claiming — FEASIBLE

**What it needs:**
- `assign_issue(issue_id, identity)` tracker operation
- `unassign_issue(issue_id)` tracker operation
- `fetch_issue(issue_id)` for post-assignment verification
- Optimistic concurrency: assign → re-fetch → verify → release if race lost

**Feasibility:** Linear's API supports `issueUpdate` mutation with `assigneeId` — this is a single GraphQL call. The verify step is just another `issue` query. The whole claim flow is 2-3 API calls.

**Risk:** Low. The only subtlety is the race window between assign and verify. TEMPAD handles this correctly with the "re-fetch and check" pattern.

**Note:** Symphony's Elixir code already has `assigned_to_worker` routing logic. TEMPAD just formalizes it as a first-class claim mechanism.

### 3.3 Split Configuration (Repo + User) — FEASIBLE

**What it needs:**
- YAML parser for `~/.tempad/config.yaml`
- Merge logic: CLI > user config > repo config > env vars > defaults
- Clear ownership rules (user owns IDE/agent/identity, repo owns hooks/states/workspace)

**Feasibility:** Trivial. Every language has YAML parsing. Merge precedence is a straightforward chain of `get_or_default` calls.

**Risk:** None.

### 3.4 `tempad init` / `tempad validate` / `tempad clean` — FEASIBLE

**What they need:**
- `init`: Write a default `~/.tempad/config.yaml` template
- `validate`: Load config + workflow, run preflight checks, print results
- `clean`: Query tracker for terminal issues, remove matching workspace dirs

**Feasibility:** These are simple CLI subcommands. `clean` reuses the existing startup terminal cleanup logic.

**Risk:** None.

### 3.5 Four Prompt Delivery Methods — FEASIBLE

**What they need:**
- `file`: Write to `<workspace>/PROMPT.md`, pass path as arg (already similar to Symphony)
- `stdin`: Pipe rendered prompt to subprocess stdin
- `arg`: Pass as CLI argument (beware shell escaping / ARG_MAX limits)
- `env`: Set `TEMPAD_PROMPT` environment variable (same ARG_MAX concern)

**Feasibility:** All trivially implementable. `file` is the safest default for large prompts.

**Risk:** `arg` and `env` can hit OS limits for very large prompts (Linux ARG_MAX ~2MB). The spec correctly defaults to `file`. Consider documenting the size limitation for `arg` and `env`.

---

## 4. Implementation Complexity Assessment

| Component | Complexity | Notes |
|---|---|---|
| Workflow Loader | Low | YAML front matter + markdown body parse. Identical to Symphony. |
| Config Layer | Low | Merge chain + typed getters. TEMPAD adds user config file but same pattern. |
| Issue Tracker Client | Medium | Linear GraphQL + pagination + normalization. TEMPAD adds assign/unassign/fetch_issue. |
| Orchestrator (Daemon) | High | Poll loop, dispatch, reconciliation, retry, concurrency — but this is identical to Symphony's orchestrator. Proven pattern. |
| Orchestrator (TUI) | Medium | Simpler than daemon — no retry, no reconciliation, just present → select → claim → workspace → IDE. |
| Workspace Manager | Low | Directory creation, hook execution, path sanitization. Identical to Symphony. |
| Agent Launcher | Low-Medium | Subprocess management + prompt delivery. Simpler than Symphony's Codex protocol integration. |
| TUI Rendering | Medium | Terminal UI framework needed. Standard library problem, well-solved. |
| Prompt Builder | Low | Liquid-compatible template rendering. Identical to Symphony. |
| HTTP Server Extension | Medium | REST API + optional dashboard. Identical to Symphony. |
| CLI | Low | Argument parsing + mode selection. Standard. |
| Logging | Low | Structured logging with issue context. Standard. |
| Dynamic Config Reload | Medium | File watcher + safe reload. Identical to Symphony. |

**Total estimated complexity: Lower than Symphony.** The Codex protocol integration (app_server.ex) is the most complex part of Symphony's implementation, and TEMPAD replaces it with simple subprocess management.

---

## 5. Risks and Open Questions

### 5.1 Multi-Turn Agent Efficiency

**Question:** Is subprocess-per-turn efficient enough for agents that benefit from persistent context (warm caches, loaded models)?

**Answer:** Yes, for the target agents. Claude Code, Codex CLI, and OpenCode all support resumption from workspace state. The 1s continuation delay is negligible. Agents that truly need persistent sessions (like Codex app-server) can manage their own turn loop internally and exit when done.

### 5.2 Linear API Rate Limits

**Question:** Does the claiming flow (assign + verify + unassign on failure) create rate limit pressure?

**Answer:** Minimal. Linear's rate limit is 1500 requests/hour for API keys. Even aggressive claiming (50 issues/tick × 3 calls each) is ~150 calls per 30s tick, which is sustainable. In practice, teams won't have 50 unassigned issues per tick.

### 5.3 Prompt Size Limits

**Question:** Can `arg` and `env` prompt delivery handle realistic prompts?

**Answer:** Linux ARG_MAX is ~2MB total for all env vars + args. A typical TEMPAD prompt with issue context is 1-10KB. Safe for most cases, but `file` is the right default. Consider adding a warning if prompt exceeds 100KB for `arg`/`env` delivery.

### 5.4 TUI + Daemon Mode Exclusivity

**Question:** Could someone want TUI and daemon simultaneously?

**Answer:** The spec says they're mutually exclusive, which is correct for v1. A future "TUI dashboard for daemon mode" could be added later without changing the core spec.

---

## 6. Summary

| Dimension | Status |
|---|---|
| Symphony feature coverage | **Complete** — all orchestration features present |
| Intentional removals | **Correct** — only Codex-specific coupling removed |
| New features feasible | **Yes** — TUI, claiming, split config, CLI subcommands all straightforward |
| Implementation complexity | **Lower** than Symphony (no Codex protocol) |
| Spec gaps to fix | **Minor** — tighten assignee filtering language (Section 2.1) |
| Blocking risks | **None identified** |

**Verdict: The TEMPAD spec is ready for architecture and tech stack decisions.** It's a clean superset of Symphony with no missing capabilities and no infeasible requirements.
