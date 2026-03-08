# T.E.M.P.A.D

**Temporal Execution & Management Poll-Agent Dispatcher**

An enhanced open-source alternative to [OpenAI's Symphony](https://github.com/openai/symphony). TEMPAD continuously reads work from Linear, presents available tasks to the developer, and either opens an IDE session (TUI mode) or runs coding agents headlessly (daemon mode) in isolated per-issue workspaces.

> [!WARNING]
> TEMPAD is in active development. Phase 1 (Foundation) is complete.

## Repository Structure

```
tempad/                                  ← monorepo root
├── docs/                                ← language-agnostic specification & design docs
│   ├── SPEC_v1.md                       ← behavioral spec — source of truth for WHAT
│   ├── STACK_COMPARISON_v1.md           ← Go vs Rust vs Elixir decision
│   ├── TEMPAD_vs_SYMPHONY_v1.md         ← feature comparison with Symphony
│   └── GAP_ANALYSIS_v1.md              ← coverage analysis vs Symphony
├── code/
│   └── go/                              ← Go implementation (self-contained)
│       ├── cmd/tempad/                  ← CLI entry points (Cobra)
│       ├── internal/                    ← all Go packages
│       ├── docs/                        ← Go-specific docs
│       │   ├── ARCHITECTURE_GO_v1.md    ← how to build it (Go-specific)
│       │   ├── PRODUCT_BACKLOG_v1.md    ← 57 tickets, 8 phases
│       │   └── BACKLOG_v1.md            ← condensed ticket list
│       ├── kanban/                      ← file-based kanban board
│       ├── README.md                    ← Go dev guide
│       ├── AGENTS.md                    ← context for coding agents
│       ├── CLAUDE.md                    ← points to AGENTS.md
│       └── go.mod
└── README.md                            ← this file
```

---

## Specification & Design

Language-agnostic documentation lives in `docs/` at the monorepo root. These define **what** TEMPAD is and **why** it exists — independent of any implementation.

| Document | Version | Purpose |
|----------|---------|---------|
| [`docs/SPEC_v1.md`](docs/SPEC_v1.md) | 1.0.0 | Behavioral specification — source of truth for **what** TEMPAD does |
| [`docs/TEMPAD_vs_SYMPHONY_v1.md`](docs/TEMPAD_vs_SYMPHONY_v1.md) | 1.0.0 | Feature comparison with Symphony |
| [`docs/GAP_ANALYSIS_v1.md`](docs/GAP_ANALYSIS_v1.md) | 1.0.0 | Coverage analysis vs Symphony |
| [`docs/STACK_COMPARISON_v1.md`](docs/STACK_COMPARISON_v1.md) | 1.0.0 | Go vs Rust vs Elixir weighted comparison |

---

## Implementations

Each implementation under `code/` is a self-contained project with its own README, agent context files, kanban board, and backlog. Implementation-specific docs (architecture, backlog) live within each implementation directory.

| Language | Path | Status | Architecture |
|----------|------|--------|-------------|
| **Go** | [`code/go/`](code/go/) | Active — Phase 1 complete | [`ARCHITECTURE_GO_v1.md`](code/go/docs/ARCHITECTURE_GO_v1.md) |

### Adding a New Implementation

To add a new language implementation:

1. Create `code/<language>/` with its own `README.md`, `AGENTS.md`, and `CLAUDE.md`
2. Add implementation-specific docs under `code/<language>/docs/` (architecture, backlog)
3. Set up a `kanban/` board for task tracking
4. Reference the shared spec (`docs/SPEC_v1.md`) as the source of truth for behavior

---

## Getting Started

See [`code/go/README.md`](code/go/README.md) for the Go implementation quick start, package layout, and development workflow.

## For Coding Agents

Each implementation has its own `AGENTS.md` and `CLAUDE.md` at its root. For the Go implementation, see [`code/go/AGENTS.md`](code/go/AGENTS.md).

## License

MIT — [OneNeural](https://github.com/oneneural)
