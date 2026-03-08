# T.E.M.P.A.D

**Temporal Execution & Management Poll-Agent Dispatcher**

An enhanced open-source alternative to [OpenAI's Symphony](https://github.com/openai/symphony). TEMPAD continuously reads work from Linear, presents available tasks to the developer, and either opens an IDE session (TUI mode) or runs coding agents headlessly (daemon mode) in isolated per-issue workspaces.

> [!WARNING]
> TEMPAD is in active development. Phase 1 (Foundation) is complete.

## Repository Structure

```
tempad/                                  ← monorepo root
├── code/
│   └── go/                              ← Go implementation (self-contained)
│       ├── cmd/tempad/                  ← CLI entry points (Cobra)
│       ├── internal/                    ← All Go packages
│       ├── docs/                        ← Go-specific docs
│       │   ├── ARCHITECTURE_GO_v1.md    ← How to build it (Go-specific)
│       │   ├── PRODUCT_BACKLOG_v1.md    ← 57 tickets, 8 phases
│       │   └── BACKLOG_v1.md            ← Condensed ticket list
│       ├── kanban/                      ← File-based kanban board
│       ├── README.md                    ← Go dev guide
│       ├── AGENTS.md                    ← Context for coding agents
│       ├── CLAUDE.md                    ← Points to AGENTS.md
│       └── go.mod
├── docs/                                ← Language-agnostic documentation
│   ├── SPEC_v1.md                       ← What to build (behavioral spec)
│   ├── STACK_COMPARISON_v1.md           ← Go vs Rust vs Elixir decision
│   ├── TEMPAD_vs_SYMPHONY_v1.md         ← Feature comparison with Symphony
│   └── GAP_ANALYSIS_v1.md              ← Coverage analysis vs Symphony
└── README.md                            ← This file
```

Each implementation under `code/` is a self-contained project with its own README, agent context files, kanban board, and backlog. Language-agnostic documentation lives in `docs/` at the monorepo root.

## Implementations

| Language | Path | Status |
|----------|------|--------|
| **Go** | [`code/go/`](code/go/) | Active — Phase 1 complete |

## Documentation

| Document | Version | Scope | Purpose |
|----------|---------|-------|---------|
| [`docs/SPEC_v1.md`](docs/SPEC_v1.md) | 1.0.0 | All | Behavioral specification — source of truth for **what** |
| [`docs/STACK_COMPARISON_v1.md`](docs/STACK_COMPARISON_v1.md) | 1.0.0 | All | Go vs Rust vs Elixir weighted comparison |
| [`docs/TEMPAD_vs_SYMPHONY_v1.md`](docs/TEMPAD_vs_SYMPHONY_v1.md) | 1.0.0 | All | Feature comparison with Symphony |
| [`docs/GAP_ANALYSIS_v1.md`](docs/GAP_ANALYSIS_v1.md) | 1.0.0 | All | Coverage analysis vs Symphony |
| [`code/go/docs/ARCHITECTURE_GO_v1.md`](code/go/docs/ARCHITECTURE_GO_v1.md) | 1.0.0 | Go | Go implementation guide — source of truth for **how** |
| [`code/go/docs/PRODUCT_BACKLOG_v1.md`](code/go/docs/PRODUCT_BACKLOG_v1.md) | 1.0.0 | Go | All 57 tickets with work items and acceptance criteria |

## Getting Started

See [`code/go/README.md`](code/go/README.md) for the Go implementation quick start, package layout, and development workflow.

## For Coding Agents

Each implementation has its own `AGENTS.md` and `CLAUDE.md` at its root. For the Go implementation, see [`code/go/AGENTS.md`](code/go/AGENTS.md).

## License

MIT — [OneNeural](https://github.com/oneneural)
