# TEMPAD — Go Implementation

[![CI](https://github.com/oneneural/tempad/actions/workflows/ci.yml/badge.svg)](https://github.com/oneneural/tempad/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/oneneural/tempad)](https://goreportcard.com/report/github.com/oneneural/tempad)
[![Release](https://img.shields.io/github/v/release/oneneural/tempad?filter=go/*&label=Latest)](https://github.com/oneneural/tempad/releases)

**Temporal Execution & Management Poll-Agent Dispatcher**

A developer-local service that polls [Linear](https://linear.app) for work, presents tasks via an interactive TUI, and dispatches coding agents in isolated per-issue workspaces. An enhanced open-source alternative to [OpenAI's Symphony](https://github.com/openai/symphony).

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Module** | `github.com/oneneural/tempad` |
| **Go** | 1.22+ |
| **Spec** | [`docs/SPEC_v1.md`](../../docs/SPEC_v1.md) v1.0.0 |
| **Architecture** | [`docs/ARCHITECTURE_GO_v1.md`](./docs/ARCHITECTURE_GO_v1.md) v1.0.0 |

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Setup Guide](#setup-guide)
- [Usage](#usage)
- [Configuration Reference](#configuration-reference)
- [Features](#features)
- [HTTP API](#http-api)
- [Package Layout](#package-layout)
- [Architecture Highlights](#architecture-highlights)
- [Testing](#testing)
- [Contributing](#contributing)

---

## Prerequisites

- **Linear account** with an API key — create at [Linear Settings → API](https://linear.app/settings/api)
- A Linear project with issues to work on

## Installation

### Homebrew (macOS/Linux)

```bash
brew install oneneural/tap/tempad
```

### Script (macOS/Linux)

```bash
curl -sSL https://raw.githubusercontent.com/oneneural/tempad/main/scripts/install.sh | bash
```

Downloads the latest release, verifies the SHA256 checksum, and installs to `/usr/local/bin`.

### Go install

```bash
go install github.com/oneneural/tempad/cmd/tempad@latest
```

Requires Go 1.22+.

### Download binary

Download the archive for your platform from the [latest release](https://github.com/oneneural/tempad/releases), extract, and add to your PATH.

### From source

```bash
git clone https://github.com/oneneural/tempad.git
cd tempad/code/go

go build -o tempad ./cmd/tempad
./tempad --help
```

---

## Setup Guide

### Step 1: Initialize configuration

```bash
tempad init
```

This creates `~/.tempad/config.yaml` with sensible defaults.

### Step 2: Add your Linear credentials

Edit `~/.tempad/config.yaml`:

```yaml
tracker:
  identity: "you@company.com"     # Your Linear email
  api_key: "$LINEAR_API_KEY"      # Uses env var (recommended)
  # api_key: "lin_api_..."        # Or hardcode (not recommended)

ide:
  command: "code"                 # Your editor: code, cursor, zed, idea, webstorm
  args: "--new-window"            # Optional extra args
```

Set the environment variable:

```bash
export LINEAR_API_KEY="lin_api_your_key_here"
```

### Step 3: Create a WORKFLOW.md in your project

Place this at the root of the repository you want TEMPAD to manage:

```markdown
---
tracker:
  kind: linear
  project_slug: "my-project"     # Your Linear project slug (from URL)
  api_key: "$LINEAR_API_KEY"

agent:
  command: "claude-code"          # Any CLI agent: claude-code, codex, aider, etc.
  prompt_delivery: "file"         # file, env, stdin, or arg
  max_concurrent: 3
  max_retries: 5

workspace:
  root: "/tmp/tempad_workspaces"

hooks:
  after_create: "git clone $REPO_URL ."
  before_run: "npm install"

terminal_states:
  - Done
  - Cancelled
---

You are working on {{issue.identifier}}: {{issue.title}}.

## Description
{{issue.description}}

## Labels
{{issue.labels | join: ", "}}
```

The YAML front matter configures TEMPAD. Everything below the `---` is the **prompt template** rendered with [Liquid](https://shopify.github.io/liquid/) syntax and passed to your agent.

### Step 4: Validate your setup

```bash
tempad validate
```

This checks all config sources (CLI, user, workflow, env) and reports any errors.

### Step 5: Run TEMPAD

```bash
# TUI mode — interactive task board
tempad --workflow ./WORKFLOW.md

# Daemon mode — headless agent dispatch
tempad --daemon --workflow ./WORKFLOW.md

# Daemon with HTTP dashboard
tempad --daemon --port 8080 --workflow ./WORKFLOW.md
```

---

## Usage

### TUI Mode (default)

Interactive terminal task board. Browse Linear issues, claim tasks, open your IDE.

```bash
tempad [--workflow WORKFLOW.md] [--identity user@example.com] [--ide code]
```

**Keybindings:**

| Key | Action |
| --- | --- |
| `j` / `k` | Navigate up / down |
| `Enter` | Claim issue + open IDE in workspace |
| `d` | Toggle detail view |
| `r` | Refresh issues from Linear |
| `u` | Release (unclaim) selected issue |
| `o` | Open issue URL in browser |
| `q` | Quit |

**What happens when you press Enter:**

1. TEMPAD assigns the issue to you on Linear
2. Creates an isolated workspace directory
3. Runs `after_create` and `before_run` hooks
4. Opens your IDE in the workspace

### Daemon Mode

Headless agent orchestrator. Polls Linear for work, claims issues, dispatches coding agents, handles retries.

```bash
tempad --daemon [--workflow WORKFLOW.md] [--agent "claude-code"] [--port 8080]
```

**Lifecycle:**

1. Polls Linear for issues in `active_states`
2. Filters out blocked issues and already-claimed issues
3. Claims up to `max_concurrent` issues
4. Creates workspaces and runs hooks
5. Renders prompt template and launches agent subprocess
6. Monitors agent output for stalls
7. On agent exit: continuation retry (exit 0) or backoff retry (exit non-zero)
8. On SIGINT/SIGTERM: cancels agents, releases all claims, exits cleanly

### Subcommands

| Command | Description |
| --- | --- |
| `tempad init` | Scaffold `~/.tempad/config.yaml` with defaults |
| `tempad validate` | Validate config from all sources, report errors |
| `tempad clean` | Remove workspace directories for terminal issues |

### CLI Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--workflow` | `./WORKFLOW.md` | Path to workflow definition file |
| `--daemon` | `false` | Run in headless daemon mode |
| `--port` | — | Start HTTP dashboard on this port (daemon only) |
| `--identity` | — | Override tracker identity (email) |
| `--ide` | — | IDE command override |
| `--agent` | — | Agent command override |
| `--log-level` | `info` | Log level: debug, info, warn, error |

---

## Configuration Reference

### Source Precedence

```text
CLI flags  >  User config  >  Workflow front matter  >  Env vars  >  Defaults
              (~/.tempad/     (WORKFLOW.md)             ($VAR)
               config.yaml)
```

### Tracker Settings

| Field | Default | Description |
| --- | --- | --- |
| `tracker.kind` | — | Tracker type (currently `linear`) |
| `tracker.endpoint` | `https://api.linear.app/graphql` | API endpoint |
| `tracker.api_key` | — | API key or `$ENV_VAR` reference |
| `tracker.project_slug` | — | Project slug (from Linear URL) |
| `tracker.identity` | — | Your email in the tracker |
| `tracker.active_states` | `["Todo", "In Progress"]` | States to poll for |
| `tracker.terminal_states` | `["Closed", "Cancelled", "Canceled", "Duplicate", "Done"]` | States considered finished |

### Polling

| Field | Default | Description |
| --- | --- | --- |
| `polling.interval_ms` | `30000` | Poll interval in milliseconds |

### Workspace

| Field | Default | Description |
| --- | --- | --- |
| `workspace.root` | `<tmp>/tempad_workspaces` | Base directory for per-issue workspaces |

### Hooks

Shell commands run at workspace lifecycle events. Executed via `bash -lc` with process group isolation.

| Field | Default | Description |
| --- | --- | --- |
| `hooks.after_create` | — | Run after workspace directory is created |
| `hooks.before_run` | — | Run before agent is launched |
| `hooks.after_run` | — | Run after agent completes |
| `hooks.before_remove` | — | Run before workspace is cleaned up |
| `hooks.timeout_ms` | `60000` | Max time per hook execution |

### Agent (daemon mode)

| Field | Default | Description |
| --- | --- | --- |
| `agent.command` | — | Shell command to run the agent |
| `agent.args` | — | Additional arguments |
| `agent.prompt_delivery` | `file` | How to pass prompt: `file`, `env`, `stdin`, `arg` |
| `agent.max_concurrent` | `5` | Max simultaneous agents |
| `agent.max_concurrent_by_state` | `{}` | Per-state concurrency limits |
| `agent.max_retries` | `10` | Max failure retries per issue |
| `agent.max_retry_backoff_ms` | `300000` | Max backoff delay (5 min) |
| `agent.max_turns` | `20` | Max continuation retries |
| `agent.turn_timeout_ms` | `3600000` | Max time per agent run (1 hour) |
| `agent.stall_timeout_ms` | `300000` | Cancel if no output for this long (5 min) |
| `agent.read_timeout_ms` | `5000` | Output read timeout |

### IDE (TUI mode)

| Field | Default | Description |
| --- | --- | --- |
| `ide.command` | `code` | IDE command (code, cursor, zed, idea, webstorm) |
| `ide.args` | — | Extra arguments |

### Display

| Field | Default | Description |
| --- | --- | --- |
| `display.theme` | `auto` | Theme: auto, dark, light |

### Example: Full WORKFLOW.md

```yaml
---
tracker:
  kind: linear
  project_slug: "backend"
  api_key: "$LINEAR_API_KEY"
  active_states:
    - Todo
    - "In Progress"
  terminal_states:
    - Done
    - Cancelled

polling:
  interval_ms: 15000

workspace:
  root: "/home/dev/workspaces"

hooks:
  after_create: |
    git clone --depth 1 git@github.com:myorg/backend.git .
    cp /home/dev/.env.template .env
  before_run: "npm install && npm run build"
  after_run: "npm test"
  timeout_ms: 120000

agent:
  command: "claude-code"
  prompt_delivery: file
  max_concurrent: 3
  max_retries: 5
  stall_timeout_ms: 600000
---

You are a senior engineer working on {{issue.identifier}}: {{issue.title}}.

## Task
{{issue.description}}

## Constraints
- Write tests for any new functionality
- Follow existing code conventions
- Do not modify unrelated files
```

### Example: User config (~/.tempad/config.yaml)

```yaml
tracker:
  identity: "dev@company.com"
  api_key: "$LINEAR_API_KEY"

ide:
  command: "cursor"
  args: "--new-window"

agent:
  command: "claude-code --auto"

display:
  theme: dark
```

---

## Features

### Core

- **Linear integration** — polls project issues via GraphQL, handles pagination and blocker detection
- **Issue claiming** — assign → fetch → verify pattern prevents race conditions across team members
- **Workspace isolation** — per-issue directories with lifecycle hooks (prepare/clean)
- **Liquid templates** — agent prompts rendered with full issue context

### TUI Mode

- **Interactive board** — Bubble Tea-based terminal UI with issue list and detail views
- **IDE integration** — claim an issue and open it directly in your editor
- **Live refresh** — poll for updates without leaving the terminal
- **Hot reload** — WORKFLOW.md changes applied instantly without restart

### Daemon Mode

- **Concurrent agents** — dispatch multiple coding agents with configurable concurrency
- **Retry strategies** — exponential backoff for failures, continuation retries for clean exits
- **Stall detection** — monitor agent stdout/stderr, cancel unresponsive workers
- **Process group isolation** — clean termination of agent subprocesses via `kill(-pgid)`
- **Graceful shutdown** — release all claims on SIGINT/SIGTERM

### HTTP Dashboard

- **REST API** — query orchestrator state, issue details, trigger manual refreshes
- **HTML dashboard** — live view of running agents and retry queue
- **Loopback only** — binds to 127.0.0.1 for security

### Observability

- **Structured logging** — slog with contextual fields per issue
- **Log rotation** — lumberjack-based rotation (50MB, 5 backups)
- **Per-issue logs** — separate log files for each agent run

---

## HTTP API

Available in daemon mode with `--port`. Binds to `127.0.0.1` only.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | HTML dashboard with running agents and retry queue |
| `GET` | `/healthz` | Health check — returns `{"status": "ok"}` |
| `GET` | `/api/v1/state` | Full orchestrator state as JSON |
| `GET` | `/api/v1/{identifier}` | Single issue details by identifier (e.g., `PROJ-123`) |
| `POST` | `/api/v1/refresh` | Trigger an immediate poll cycle |

### Example

```bash
# Check health
curl http://localhost:8080/healthz

# Get orchestrator state
curl http://localhost:8080/api/v1/state | jq

# Get specific issue
curl http://localhost:8080/api/v1/PROJ-123 | jq

# Force refresh
curl -X POST http://localhost:8080/api/v1/refresh
```

---

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
| [`spf13/cobra`](https://github.com/spf13/cobra) | CLI framework |
| [`osteele/liquid`](https://github.com/osteele/liquid) | Liquid template engine |
| [`gopkg.in/yaml.v3`](https://github.com/go-yaml/yaml) | YAML parsing |
| [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm Architecture) |
| [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) | TUI styling |
| [`go-chi/chi/v5`](https://github.com/go-chi/chi) | HTTP router |
| [`fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify) | File watching for hot reload |
| [`natefinch/lumberjack`](https://github.com/natefinch/lumberjack) | Log file rotation |
| [`stretchr/testify`](https://github.com/stretchr/testify) | Test assertions |
| [`uber-go/goleak`](https://github.com/uber-go/goleak) | Goroutine leak detection (test only) |

---

## Architecture Highlights

- **Orchestrator for-select loop** — single goroutine owns all mutable state, communicates via buffered channels
- **Process group isolation** — agent subprocesses run in their own process group for clean termination
- **Stall detection** — atomic timestamps track agent output, cancel stalled workers
- **Exponential backoff** — `min(10000 * 2^(attempt-1), max_retry_backoff_ms)` for failure retries
- **Continuation retries** — exit-0 agents get re-dispatched after 1s (don't count toward max_retries)
- **Hot reload** — fsnotify watches WORKFLOW.md with 500ms debounce, applies config without restarting agents
- **Claim verification** — assign → fetch → verify pattern prevents race conditions

See [`docs/ARCHITECTURE_GO_v1.md`](./docs/ARCHITECTURE_GO_v1.md) for the full architecture document.

---

## Testing

```bash
# Build
go build ./cmd/tempad

# Unit tests
go test ./...

# Race detector + goroutine leak detection
go test -race ./...

# Vet
go vet ./...

# Smoke tests (requires real Linear credentials)
LINEAR_API_KEY=lin_api_... LINEAR_TEST_PROJECT_SLUG=my-project go test -tags smoke ./test/...
```

### Test Coverage

| Package | Coverage |
| --- | --- |
| internal/claim | 100% |
| internal/tracker | 100% |
| internal/logging | 100% |
| internal/tracker/linear | 88% |
| internal/agent | 87% |
| internal/workspace | 85% |
| internal/config | 79% |
| internal/orchestrator | 77% |
| internal/prompt | 77% |
| internal/server | 74% |
| internal/domain | ~70% |

---

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

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

---

## Build Status

| Phase | Description | Status |
| --- | --- | --- |
| 1 | Foundation (CLI, config, domain, prompt) | COMPLETE (10/10) |
| 2 | Tracker Client (Linear GraphQL) | COMPLETE (6/6) |
| 3 | Workspace Manager (paths, hooks, cleanup) | COMPLETE (5/5) |
| 4 | TUI Mode (Bubble Tea board + detail views) | COMPLETE (9/9) |
| 5 | Daemon Mode (orchestrator, dispatch, retry) | COMPLETE (14/14) |
| 6 | Hot Reload + Logging (fsnotify, slog, lumberjack) | COMPLETE (4/4) |
| 7 | HTTP Server (Chi, REST API, dashboard) | COMPLETE (3/3) |
| 8 | Testing + Hardening (race, leaks, e2e, smoke) | COMPLETE (6/6) |

**Total: 57/57 tickets complete.**

## License

MIT — [OneNeural](https://github.com/oneneural)
