# TEMPAD — Go Implementation

**Temporal Execution & Management Poll-Agent Dispatcher**

A developer-local service that polls Linear for work, presents tasks via an interactive TUI, and dispatches coding agents in isolated per-issue workspaces.

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Module** | `github.com/oneneural/tempad` |
| **Go** | 1.22+ |
| **Spec** | [`docs/SPEC_v1.md`](../../docs/SPEC_v1.md) v1.0.0 |
| **Architecture** | [`docs/ARCHITECTURE_GO_v1.md`](./docs/ARCHITECTURE_GO_v1.md) v1.0.0 |

## Prerequisites

- **Go 1.22+** — [install](https://go.dev/dl/)
- **Linear API key** — create at [Linear Settings → API](https://linear.app/settings/api)
- A Linear project with issues to work on

## Installation

```bash
# Clone the repository
git clone https://github.com/oneneural/tempad.git
cd tempad/code/go

# Build the binary
go build -o tempad ./cmd/tempad

# Verify installation
./tempad --help
```

## Getting Started

### 1. Initialize configuration

```bash
./tempad init
```

This creates `~/.tempad/config.yaml` with default settings. Edit it to add your Linear API key:

```yaml
tracker:
  kind: linear
  api_key: "lin_api_..."       # Your Linear API key
  project_slug: "my-project"   # Your Linear project slug
  identity: "user@example.com" # Your Linear email
```

### 2. Create a WORKFLOW.md (optional)

The workflow file defines per-repo agent settings. Place it at the root of your project:

```markdown
---
agent:
  command: "claude-code"
  prompt_delivery: "file"
terminal_states:
  - Done
  - Cancelled
---

You are working on {{issue.identifier}}: {{issue.title}}
```

### 3. Run in TUI mode

```bash
./tempad --workflow path/to/WORKFLOW.md
```

### 4. Run in daemon mode

```bash
# Headless agent dispatch
./tempad --daemon --workflow path/to/WORKFLOW.md

# With HTTP dashboard
./tempad --daemon --port 8080 --workflow path/to/WORKFLOW.md
```

### 5. Validate your configuration

```bash
./tempad validate
```

## Usage

### TUI Mode (default)

Interactive terminal task board. Browse issues, claim tasks, open your IDE.

```
tempad [--workflow WORKFLOW.md] [--identity user@example.com] [--ide code]
```

**Keybindings:**

| Key | Action |
| --- | --- |
| `j` / `k` | Navigate up/down |
| `Enter` | Claim issue + open IDE |
| `d` | Detail view |
| `r` | Refresh issues |
| `u` | Release claimed issue |
| `o` | Open issue URL in browser |
| `q` | Quit |

### Daemon Mode

Headless agent orchestrator. Polls for work, dispatches coding agents, handles retries and stall detection.

```
tempad --daemon [--workflow WORKFLOW.md] [--agent "claude-code"] [--port 8080]
```

**Features:**
- Automatic issue claiming and agent dispatch
- Configurable concurrency (`max_concurrent`)
- Exponential backoff retries (`max_retries`)
- Stall detection with automatic cancellation
- Continuation retries for agents that exit cleanly
- Hot reload of WORKFLOW.md without restarting

### Subcommands

| Command | Description |
| --- | --- |
| `tempad init` | Scaffold `~/.tempad/config.yaml` with defaults |
| `tempad validate` | Check config for errors and report issues |
| `tempad clean` | Remove terminal workspace directories |

### CLI Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--workflow` | `./WORKFLOW.md` | Path to workflow definition file |
| `--daemon` | `false` | Run in headless daemon mode |
| `--port` | — | Start HTTP dashboard on this port (daemon only) |
| `--identity` | — | Override tracker identity (email) |
| `--ide` | — | IDE to open when claiming issues |
| `--agent` | — | Agent command override |

### Configuration

5-level merge with clear precedence:

```
CLI flags > User (~/.tempad/config.yaml) > Repo (WORKFLOW.md front matter) > Env ($VAR) > Defaults
```

See the [spec](../../docs/SPEC_v1.md) Section 8 for full config reference.

## Features

### Core
- **Linear integration** — polls project issues with GraphQL, handles pagination and blocker detection
- **Issue claiming** — assign → fetch → verify pattern prevents race conditions across team members
- **Workspace isolation** — per-issue directories with lifecycle hooks (prepare/clean)
- **Liquid templates** — agent prompts rendered with issue context using [Liquid](https://shopify.github.io/liquid/) syntax

### TUI Mode
- **Interactive board** — Bubble Tea-based terminal UI with issue list and detail views
- **IDE integration** — claim an issue and open it in your preferred editor
- **Live refresh** — poll for updates without leaving the terminal
- **Hot reload** — WORKFLOW.md changes applied without restarting

### Daemon Mode
- **Concurrent agents** — dispatch multiple coding agents with configurable concurrency limits
- **Retry strategies** — exponential backoff for failures, continuation retries for clean exits
- **Stall detection** — monitor agent output and cancel stalled workers
- **Process group isolation** — clean termination of agent subprocesses
- **Graceful shutdown** — release all claims on SIGINT/SIGTERM

### HTTP Dashboard
- **REST API** — query orchestrator state, issue details, trigger refreshes
- **HTML dashboard** — live view of running agents and retry queue
- **Loopback only** — binds to 127.0.0.1 for security

### Observability
- **Structured logging** — slog with contextual fields per issue
- **Log rotation** — lumberjack-based rotation for daemon log files
- **Per-issue logs** — separate log files for each agent run

## HTTP API

When running in daemon mode with `--port`, the following endpoints are available on `127.0.0.1`:

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | HTML dashboard |
| `GET` | `/healthz` | Health check (`{"status": "ok"}`) |
| `GET` | `/api/v1/state` | Full orchestrator state (JSON) |
| `GET` | `/api/v1/{identifier}` | Issue details by identifier |
| `POST` | `/api/v1/refresh` | Trigger immediate poll cycle |

## Package Layout

```text
cmd/tempad/              CLI entry points (Cobra)
├── main.go              Root command — TUI vs daemon mode switch
├── init.go              tempad init — scaffold ~/.tempad/config.yaml
├── validate.go          tempad validate — check config
└── clean.go             tempad clean — workspace cleanup

internal/
├── domain/              Core types: Issue, Workspace, Run, OrchestratorState
├── config/              5-level config merge, validation, file watcher
├── tracker/             tracker.Client interface + typed errors
│   └── linear/          Linear GraphQL: queries, pagination, normalization
├── workspace/           Path resolution, hooks (bash -lc), Prepare/Clean lifecycle
├── prompt/              Liquid templates (osteele/liquid) for agent prompts
├── agent/               Subprocess launcher, process groups, prompt delivery
├── claim/               Stateless claim: assign → fetch → verify → release
├── orchestrator/        Daemon: select loop, dispatch, workers, retry, reconciliation
├── tui/                 Bubble Tea: app model, board view, detail view, keybindings
├── server/              Chi HTTP: REST API + HTML dashboard (loopback only)
└── logging/             slog + lumberjack rotation, per-issue agent logs

test/                    E2E and smoke tests
```

## Dependencies

| Package | Purpose |
| --- | --- |
| `spf13/cobra` | CLI framework |
| `osteele/liquid` | Liquid template engine |
| `gopkg.in/yaml.v3` | YAML parsing |
| `charmbracelet/bubbletea` | TUI framework (Elm Architecture) |
| `charmbracelet/lipgloss` | TUI styling |
| `charmbracelet/bubbles` | TUI components |
| `go-chi/chi/v5` | HTTP router |
| `fsnotify/fsnotify` | File watching for hot reload |
| `natefinch/lumberjack` | Log file rotation |
| `stretchr/testify` | Test assertions |
| `uber-go/goleak` | Goroutine leak detection (test only) |

## Architecture Highlights

- **Orchestrator for-select loop** — single goroutine owns all mutable state, communicates via buffered channels
- **Process group isolation** — agent subprocesses run in their own process group for clean termination
- **Stall detection** — atomic timestamps track agent output, cancel stalled workers
- **Exponential backoff** — `min(10000 * 2^(attempt-1), max_retry_backoff_ms)` for failure retries
- **Continuation retries** — exit-0 agents get re-dispatched after 1s (don't count toward max_retries)
- **Hot reload** — fsnotify watches WORKFLOW.md with 500ms debounce, applies config without restarting agents
- **Claim verification** — assign → fetch → verify pattern prevents race conditions

See [`docs/ARCHITECTURE_GO_v1.md`](./docs/ARCHITECTURE_GO_v1.md) for the full architecture document.

## Testing

```bash
go build ./cmd/tempad        # Must compile
go test ./...                # All unit tests pass
go test -race ./...          # No race conditions (includes goroutine leak detection)
go vet ./...                 # No vet warnings

# Smoke tests (requires real Linear credentials)
LINEAR_API_KEY=... LINEAR_TEST_PROJECT_SLUG=... go test -tags smoke ./test/...
```

## Development

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

- Phase 1: Foundation (CLI, config, domain, prompt) — COMPLETE (10/10)
- Phase 2: Tracker Client (Linear GraphQL) — COMPLETE (6/6)
- Phase 3: Workspace Manager (paths, hooks, cleanup) — COMPLETE (5/5)
- Phase 4: TUI Mode (Bubble Tea board + detail views) — COMPLETE (9/9)
- Phase 5: Daemon Mode (orchestrator, dispatch, retry) — COMPLETE (14/14)
- Phase 6: Hot Reload + Logging (fsnotify, slog, lumberjack) — COMPLETE (4/4)
- Phase 7: HTTP Server (Chi, REST API, dashboard) — COMPLETE (3/3)
- Phase 8: Testing + Hardening (race, leaks, e2e, smoke) — COMPLETE (6/6)

**Total: 57/57 tickets complete.**

## License

MIT — [OneNeural](https://github.com/oneneural)
