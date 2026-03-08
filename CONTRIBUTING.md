# Contributing to TEMPAD

Thank you for your interest in contributing to TEMPAD! This guide will help you get started.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Code Guidelines](#code-guidelines)
- [Commit Convention](#commit-convention)
- [Pull Requests](#pull-requests)
- [Reporting Issues](#reporting-issues)
- [Adding a New Tracker](#adding-a-new-tracker)

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/tempad.git`
3. Create a branch: `git checkout -b feat/your-feature`
4. Make your changes
5. Run tests: `go test -race ./...`
6. Submit a pull request

## Development Setup

### Prerequisites

- **Go 1.22+** — [install](https://go.dev/dl/)
- **Linear API key** (optional, only for smoke tests) — [create one](https://linear.app/settings/api)

### Build and Test

```bash
cd code/go

# Build
go build ./cmd/tempad

# Run all tests
go test ./...

# Race detector
go test -race ./...

# Vet
go vet ./...

# Run a specific package's tests
go test ./internal/config/...

# Smoke tests (requires real Linear credentials)
LINEAR_API_KEY=lin_api_... LINEAR_TEST_PROJECT_SLUG=my-project go test -tags smoke ./test/...
```

### Running Locally

```bash
# Initialize config
./tempad init

# Edit ~/.tempad/config.yaml with your Linear credentials

# TUI mode
./tempad --workflow WORKFLOW.md

# Daemon mode
./tempad --daemon --workflow WORKFLOW.md

# Daemon with HTTP dashboard
./tempad --daemon --port 8080 --workflow WORKFLOW.md
```

## Project Structure

```text
tempad/
├── docs/                          Language-agnostic spec and design docs
│   └── SPEC_v1.md                 Behavioral spec (source of truth)
├── code/go/                       Go implementation
│   ├── cmd/tempad/                CLI entry points (Cobra)
│   ├── internal/                  All packages (unexported)
│   │   ├── domain/                Core types (no dependencies)
│   │   ├── config/                Config loading, merge, validation, watcher
│   │   ├── tracker/               Tracker interface + Linear implementation
│   │   ├── workspace/             Workspace lifecycle, hooks, cleanup
│   │   ├── prompt/                Liquid template rendering
│   │   ├── agent/                 Subprocess launcher, prompt delivery
│   │   ├── claim/                 Claim mechanism (shared by TUI + daemon)
│   │   ├── orchestrator/          Daemon mode select loop
│   │   ├── tui/                   Bubble Tea interactive UI
│   │   ├── server/                Chi HTTP server + dashboard
│   │   └── logging/               slog + lumberjack rotation
│   └── test/                      E2E and smoke tests
├── CONTRIBUTING.md                This file
├── CODE_OF_CONDUCT.md             Community guidelines
├── SECURITY.md                    Security policy
└── LICENSE                        MIT
```

### Key Architectural Rule

`domain/` is a leaf node with zero imports. All other packages import `domain/` but never the reverse. This prevents circular dependencies.

## Making Changes

### Before You Start

1. Check [existing issues](https://github.com/oneneural/tempad/issues) to see if someone is already working on it
2. For significant changes, open an issue first to discuss the approach
3. Read the [spec](docs/SPEC_v1.md) for behavioral requirements and the [architecture](code/go/docs/ARCHITECTURE_GO_v1.md) for structural guidance

### What to Work On

- Bug fixes
- Documentation improvements
- Test coverage improvements
- New tracker integrations (e.g., Jira, GitHub Issues)
- Performance improvements
- New features (discuss in an issue first)

## Testing

All changes must pass:

```bash
go test ./...        # All tests pass
go test -race ./...  # No race conditions
go vet ./...         # No vet warnings
```

### Writing Tests

- **Table-driven tests** for pure functions
- **Mock interfaces** for dependencies (tracker, launcher)
- **`testify/assert`** for assertions
- **`goleak`** for goroutine leak detection in orchestrator tests
- Test files live next to the code they test (`foo.go` → `foo_test.go`)

## Code Guidelines

- **Go 1.22+** features are welcome
- **No global state** — pass dependencies through function parameters or struct fields
- **Context propagation** — all I/O operations take `context.Context` as the first parameter
- **Typed errors** — use custom error types with `errors.Is`/`errors.As` support
- **Structured logging** — use `slog` with contextual fields
- **Interfaces at boundaries** — `tracker.Client`, `agent.Launcher` are interfaces; everything else is concrete
- **`internal/`** — all packages are internal (this is a CLI, not a library)

## Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`

**Scopes:** `domain`, `config`, `tracker`, `linear`, `workspace`, `prompt`, `agent`, `claim`, `orchestrator`, `tui`, `server`, `logging`, `cli`

**Examples:**
```
feat(orchestrator): add stall detection for agent workers
fix(config): resolve env vars before validation
test(claim): add race condition coverage
docs(readme): update configuration reference
```

## Pull Requests

- Keep PRs focused on a single concern
- Include tests for new functionality
- Update documentation if behavior changes
- Ensure `go test -race ./...` passes
- Ensure `go vet ./...` is clean
- Fill out the PR template

## Reporting Issues

When reporting bugs, please include:

- Go version (`go version`)
- OS and architecture (`uname -a`)
- Steps to reproduce
- Expected vs actual behavior
- Relevant log output (from `~/.tempad/logs/`)

## Adding a New Tracker

TEMPAD is designed to support multiple issue trackers. To add one:

1. Create `internal/tracker/<name>/` (e.g., `internal/tracker/jira/`)
2. Implement the `tracker.Client` interface (6 methods)
3. Add normalization to map the tracker's data model to `domain.Issue`
4. Add a new `tracker.kind` value to config validation
5. Wire it up in `cmd/tempad/main.go`

See `internal/tracker/linear/` for the reference implementation.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
