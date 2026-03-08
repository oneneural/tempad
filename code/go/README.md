# TEMPAD — Go Implementation

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Module** | `github.com/oneneural/tempad` |
| **Go** | 1.22+ |
| **Spec** | [`docs/SPEC_v1.md`](../../docs/SPEC_v1.md) v1.0.0 |
| **Architecture** | [`docs/ARCHITECTURE_GO_v1.md`](./docs/ARCHITECTURE_GO_v1.md) v1.0.0 |

> [!WARNING]
> TEMPAD is in active development. Phase 1 (Foundation) is complete.

## Quick Start

```bash
cd code/go

# Build
go build ./cmd/tempad

# Test
go test ./...
go test -race ./...
go vet ./...
```

## Package Layout

```text
cmd/tempad/              CLI entry points (Cobra)
├── main.go              Root command — TUI vs daemon mode switch
├── init.go              tempad init — scaffold ~/.tempad/config.yaml
├── validate.go          tempad validate — check config
└── clean.go             tempad clean — workspace cleanup

internal/
├── domain/              Core types: Issue, Workspace, Run, OrchestratorState
├── config/              5-level config merge (CLI > User > Repo > Env > Defaults)
├── tracker/             tracker.Client interface + typed errors
│   └── linear/          Linear GraphQL: queries, pagination, normalization
├── workspace/           Path resolution, hooks (bash -lc), Prepare/Clean lifecycle
├── prompt/              Liquid templates (osteele/liquid) for agent prompts
├── agent/               Subprocess launcher, output monitor, prompt delivery
├── claim/               Stateless claim: assign → fetch → verify → release
├── orchestrator/        Daemon: select loop, dispatch, workers, retry, reconciliation
├── tui/                 Bubble Tea: app model, board view, detail view, keybindings
├── server/              Chi HTTP: REST API + HTML dashboard (loopback only)
└── logging/             slog + lumberjack rotation
```

## Dependencies

### Current (Phase 1)

| Package | Purpose |
| --------- | --------- |
| `spf13/cobra` | CLI framework |
| `osteele/liquid` | Liquid template engine |
| `stretchr/testify` | Test assertions |
| `gopkg.in/yaml.v3` | YAML parsing |

### Upcoming (Phases 2-8)

| Package | Purpose |
| --------- | --------- |
| `charmbracelet/bubbletea` | TUI framework (Elm Architecture) |
| `charmbracelet/lipgloss` | TUI styling |
| `charmbracelet/glamour` | Markdown rendering in TUI |
| `go-chi/chi/v5` | HTTP router |
| `go-chi/render` | JSON response helpers |
| `fsnotify/fsnotify` | File watching for hot reload |
| `natefinch/lumberjack` | Log file rotation |
| `uber-go/goleak` | Goroutine leak detection |
| `cenkalti/backoff/v4` | Exponential backoff with jitter |

## Go Conventions

- **No global state** — all state flows through function parameters or struct fields
- **Interface-first at boundaries** — `tracker.Client` is an interface so Linear is swappable
- **Typed errors** — custom error types with `errors.Is`/`errors.As` support
- **Context propagation** — all I/O operations take `context.Context` as first parameter
- **Structured logging** — `slog` with contextual fields (`slog.With("issue", id)`)
- **Table-driven tests** — for pure functions (config merge, candidate selection, backoff)

## Critical Implementation Notes

These are validated through research — do NOT use the naive approach:

1. **Linear API**: Use `project.slug` (NOT `slugId`) — `slugId` is deprecated
2. **Path safety**: Use `filepath.Rel()` or `filepath.IsLocal()`, NEVER `strings.HasPrefix()`
3. **Process groups**: `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pid, sig)` (negative PID)
4. **fsnotify**: Watch the **directory**, not the file (editors use rename-and-replace)
5. **Log rotation**: slog has NO built-in rotation — use `lumberjack` as `io.Writer`
6. **Channel buffers**: `make(chan WorkerResult, maxConcurrent)` to prevent goroutine leaks
7. **Retry timers**: `time.AfterFunc` callbacks MUST check `ctx.Err()` before state mutation
8. **Poll dedup**: `pollInFlight bool` flag in TUI to prevent overlapping polls

Full research details: [`kanban/handoffs/research-findings.md`](kanban/handoffs/research-findings.md)

## Development Workflow

1. Check the kanban board: [`kanban/`](kanban/)
2. Read the master plan for your phase: [`kanban/plans/`](kanban/plans/)
3. Pick a task from `todo/`, move it to `in-progress/`
4. Read the task file — especially the **Research Notes** section
5. Implement, test, verify acceptance criteria
6. Move to `done/`, commit with conventional message

### Commit Convention

```text
<type>(<scope>): <description>

Types:  feat, fix, refactor, test, docs, chore, ci
Scopes: domain, config, tracker, linear, workspace, prompt, agent,
        claim, orchestrator, tui, server, logging, cli
```

### Branch Naming

```text
feat/pX-YY-short-description
```

## Build Status

- Phase 1: Foundation — ✅ COMPLETE (10/10)
- Phase 2: Tracker Client — TODO (6 tickets)
- Phase 3: Workspace Manager — TODO (5 tickets)
- Phase 4: TUI Mode — TODO (9 tickets)
- Phase 5: Daemon Mode — TODO (14 tickets)
- Phase 6: Hot Reload + Logging — TODO (4 tickets)
- Phase 7: HTTP Server — TODO (3 tickets)
- Phase 8: Testing + Hardening — TODO (6 tickets)
