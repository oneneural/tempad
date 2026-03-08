# AGENTS.md — Agent Context for T.E.M.P.A.D. (Go)

This file provides context for AI coding agents (Claude Code, Cursor, Copilot, Windsurf, Cline, Aider, etc.) working on the TEMPAD Go implementation.

## What is TEMPAD?

T.E.M.P.A.D. (Temporal Execution & Management Poll-Agent Dispatcher) is a developer-local service that polls Linear for work, presents tasks via a TUI, and dispatches coding agents in isolated workspaces. It is an enhanced open-source alternative to OpenAI's Symphony.

Module: `github.com/oneneural/tempad`

## Critical Documents — Read These First

| Document | Location | Version | Purpose |
| ---------- | ---------- | --------- | --------- |
| **Spec** | [`../../docs/SPEC_v1.md`](../../docs/SPEC_v1.md) | 1.0.0 | Source of truth for **what** TEMPAD does |
| **Architecture (Go)** | [`docs/ARCHITECTURE_GO_v1.md`](docs/ARCHITECTURE_GO_v1.md) | 1.0.0 | Source of truth for **how** to build it in Go |
| **Product Backlog** | [`docs/PRODUCT_BACKLOG_v1.md`](docs/PRODUCT_BACKLOG_v1.md) | 1.0.0 | All 57 tickets with work items and acceptance criteria |
| **Kanban Board** | [`kanban/`](kanban/) | — | Current task state — todo, in-progress, done |
| **Research Findings** | [`kanban/handoffs/research-findings.md`](kanban/handoffs/research-findings.md) | — | Validated patterns, corrections, and best practices |
| **Go README** | [`README.md`](README.md) | 1.0.0 | Go-specific dev guide, packages, dependencies |

### Priority Reading Order

1. Check `kanban/in-progress/` for any active tasks
2. Read the task file for your current work (in `kanban/todo/` or `kanban/in-progress/`)
3. Read the **Research Notes** section at the bottom of that task file
4. Reference `../../docs/SPEC_v1.md` sections cited in the task's `**Spec:**` field
5. Reference `docs/ARCHITECTURE_GO_v1.md` sections cited in the task's `**Arch:**` field

## Repository Structure

```text
code/go/                                 ← Go implementation root
├── cmd/tempad/                          ← CLI (Cobra): main, init, validate, clean
├── internal/
│   ├── domain/                          ← Core types: Issue, Workspace, Run, State
│   ├── config/                          ← 5-level config merge
│   ├── tracker/                         ← tracker.Client interface + typed errors
│   │   └── linear/                      ← Linear GraphQL: queries, pagination, normalization
│   ├── workspace/                       ← Path resolution, hooks, Prepare/Clean lifecycle
│   ├── prompt/                          ← Liquid templates for agent prompts
│   ├── agent/                           ← Subprocess launcher, output monitor, prompt delivery
│   ├── claim/                           ← Stateless claim: assign → verify → release
│   ├── orchestrator/                    ← Daemon: select loop, dispatch, workers, retry
│   ├── tui/                             ← Bubble Tea: board view, detail view, keybindings
│   ├── server/                          ← Chi HTTP: REST API + dashboard (loopback)
│   └── logging/                         ← slog + lumberjack rotation
├── docs/
│   ├── ARCHITECTURE_GO_v1.md            ← How to build it (Go-specific)
│   ├── PRODUCT_BACKLOG_v1.md            ← 57 tickets, 8 phases
│   └── BACKLOG_v1.md                    ← Condensed ticket list
├── kanban/                              ← File-based kanban board
│   ├── plans/                           ← 8 master plans with research findings
│   ├── todo/                            ← Empty (all complete)
│   ├── in-progress/
│   ├── done/                            ← 57 completed (all phases)
│   ├── parked/
│   └── handoffs/                        ← research-findings.md
├── README.md
├── AGENTS.md                            ← This file
├── CLAUDE.md                            ← Points here
└── go.mod
```

## Go Conventions

- **Go 1.22+** — use modern features
- **No global state** — all state flows through parameters or struct fields
- **Interface-first at boundaries** — `tracker.Client` is swappable, test doubles work
- **Typed errors** — `errors.Is`/`errors.As` support (see `internal/tracker/errors.go`)
- **Context propagation** — all I/O takes `context.Context` first
- **Structured logging** — `slog.With("issue", id)` for contextual fields

## Critical Implementation Notes (from research)

Do NOT use the naive approach — these are validated corrections:

1. **Linear API**: Use `project.slug` (NOT `slugId`) — `slugId` is deprecated
2. **Path safety**: Use `filepath.Rel()` or `filepath.IsLocal()`, NEVER `strings.HasPrefix()`
3. **Process groups**: `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pid, sig)` (negative PID)
4. **fsnotify**: Watch the **directory**, not the file (editors use rename-and-replace)
5. **Log rotation**: slog has NO built-in rotation — use `lumberjack` as `io.Writer`
6. **Channel buffers**: `make(chan WorkerResult, maxConcurrent)` to prevent goroutine leaks
7. **Retry timers**: `time.AfterFunc` callbacks MUST check `ctx.Err()` before state mutation
8. **Poll dedup**: `pollInFlight bool` flag in TUI to prevent overlapping polls

## Config

5-level merge (see `internal/config/resolve.go`):

```text
CLI flags > User (~/.tempad/config.yaml) > Repo (WORKFLOW.md front matter) > Env ($VAR) > Defaults
```

Merged result: `ServiceConfig` struct with 33 typed fields.

## Testing

```bash
go build ./cmd/tempad        # Must compile
go test ./...                # All unit tests pass
go test -race ./...          # No race conditions
go vet ./...                 # No vet warnings
```

## Kanban Workflow

```bash
# 1. Move task to in-progress
git mv kanban/todo/pX-YY-name.md kanban/in-progress/

# 2. Read the task file — problem, solution, files, work items, acceptance criteria, research notes

# 3. Implement and test (go test -race ./...)

# 4. Move to done
git mv kanban/in-progress/pX-YY-name.md kanban/done/

# 5. Commit
git commit -m "feat(scope): description

Implements T-PXYY."
```

### Commit Convention

```text
<type>(<scope>): <description>

Types:  feat, fix, refactor, test, docs, chore, ci
Scopes: domain, config, tracker, linear, workspace, prompt, agent,
        claim, orchestrator, tui, server, logging, cli, kanban
```

### Branch Naming

```text
feat/pX-YY-short-description
```

## Task Context (TEMPAD_TASK.md)

When TEMPAD dispatches work to a workspace (via TUI or daemon mode), it writes a `TEMPAD_TASK.md` file to the workspace root. This file contains the fully rendered workflow prompt with the assigned issue's context (identifier, title, description, status, labels, blockers, etc.).

**Before starting any work in a TEMPAD workspace, check for `TEMPAD_TASK.md` and follow its instructions.** It is the primary source of truth for what you should be working on and how to execute the task.

## Rules

1. The spec (`../../docs/SPEC_v1.md`) is the source of truth for behavior. Implementation must not conflict.
2. Keep changes narrowly scoped to the current task. Avoid unrelated refactors.
3. Every task file has acceptance criteria — all must pass before marking done.
4. Read the **Research Notes** at the bottom of each task for validated patterns and gotchas.
5. Run `go build ./cmd/tempad && go test -race ./...` after every change.
6. Never run agent commands outside the workspace directory. Workspace safety is critical.
7. If behavior changes meaningfully, update the spec in the same change.
8. **No AI references in commits or PRs** — Never mention Claude, GPT, Copilot, AI, or any AI tool in git commit messages, PR titles, PR descriptions, or branch names. Write commits as if a human authored them.
9. **Documentation** — Update README when adding new packages or changing public APIs. Keep docs precise and minimal — explain only what's necessary, no bulk info.
