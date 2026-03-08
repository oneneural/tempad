# T.E.M.P.A.D.

**Temporal Execution & Management Poll-Agent Dispatcher**

An enhanced open-source alternative to [OpenAI's Symphony](https://github.com/openai/symphony). TEMPAD continuously reads work from Linear, presents available tasks to the developer, and either opens an IDE session (TUI mode) or runs coding agents headlessly (daemon mode) in isolated per-issue workspaces.

## What Does TEMPAD Do?

- **Live task board** — browse available Linear issues in an interactive terminal UI
- **One-click claim** — pick a task, claim it on Linear, and open your IDE in an isolated workspace
- **Agent dispatch** — run coding agents (Claude Code, Codex, etc.) headlessly with automatic retry
- **Workspace isolation** — per-issue directories with lifecycle hooks for setup and cleanup
- **Team coordination** — Linear assignment prevents duplicate work across team members
- **Hot reload** — change `WORKFLOW.md` and settings apply without restarting

## Getting Started

### Prerequisites

- **Go 1.22+** — [install](https://go.dev/dl/)
- **Linear API key** — create at [Linear Settings → API](https://linear.app/settings/api)

### Install

```bash
git clone https://github.com/oneneural/tempad.git
cd tempad/code/go

go build -o tempad ./cmd/tempad
./tempad --help
```

### Configure

```bash
./tempad init
```

Edit `~/.tempad/config.yaml`:

```yaml
tracker:
  kind: linear
  api_key: "lin_api_..."
  project_slug: "my-project"
  identity: "user@example.com"
```

### Run

```bash
# Interactive TUI — browse and claim tasks
./tempad --workflow WORKFLOW.md

# Headless daemon — auto-dispatch coding agents
./tempad --daemon --workflow WORKFLOW.md

# Daemon with HTTP dashboard
./tempad --daemon --port 8080 --workflow WORKFLOW.md
```

See [`code/go/README.md`](code/go/README.md) for full usage, CLI flags, features, and configuration reference.

---

## Repository Structure

```text
tempad/
├── docs/                                ← Language-agnostic specification & design
│   ├── SPEC_v1.md                       ← Behavioral spec — source of truth
│   ├── STACK_COMPARISON_v1.md           ← Go vs Rust vs Elixir decision
│   ├── TEMPAD_vs_SYMPHONY_v1.md         ← Feature comparison with Symphony
│   └── GAP_ANALYSIS_v1.md              ← Coverage analysis vs Symphony
├── code/
│   └── go/                              ← Go implementation (self-contained)
│       ├── cmd/tempad/                  ← CLI entry points (Cobra)
│       ├── internal/                    ← All Go packages
│       ├── docs/                        ← Go-specific architecture & backlog
│       ├── kanban/                      ← File-based kanban board
│       ├── README.md                    ← Go dev guide
│       └── go.mod
├── .github/
│   ├── workflows/                       ← CI, labeler, label sync
│   ├── labels.yml                       ← Label definitions (name, color, description)
│   └── labeler.yml                      ← PR auto-labeling rules
└── README.md                            ← This file
```

## Specification & Design

| Document | Purpose |
| --- | --- |
| [`docs/SPEC_v1.md`](docs/SPEC_v1.md) | Behavioral specification — source of truth for **what** TEMPAD does |
| [`docs/TEMPAD_vs_SYMPHONY_v1.md`](docs/TEMPAD_vs_SYMPHONY_v1.md) | Feature comparison with Symphony |
| [`docs/GAP_ANALYSIS_v1.md`](docs/GAP_ANALYSIS_v1.md) | Coverage analysis vs Symphony |
| [`docs/STACK_COMPARISON_v1.md`](docs/STACK_COMPARISON_v1.md) | Go vs Rust vs Elixir weighted comparison |

## Implementation Status

| Language | Path | Status | Progress | Architecture |
| --- | --- | --- | --- | --- |
| **Go** | [`code/go/`](code/go/) | Complete | 57/57 tickets | [`ARCHITECTURE_GO_v1.md`](code/go/docs/ARCHITECTURE_GO_v1.md) |

### Adding a New Implementation

1. Create `code/<language>/` with its own `README.md`, `AGENTS.md`, and `CLAUDE.md`
2. Add implementation-specific docs under `code/<language>/docs/` (architecture, backlog)
3. Set up a `kanban/` board for task tracking
4. Reference the shared spec (`docs/SPEC_v1.md`) as the source of truth for behavior

## For Coding Agents

Each implementation has its own `AGENTS.md` and `CLAUDE.md` at its root. See [`code/go/AGENTS.md`](code/go/AGENTS.md) for the Go implementation.

## License

MIT — [OneNeural](https://github.com/oneneural)
