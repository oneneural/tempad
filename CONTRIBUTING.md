# Contributing to TEMPAD

Thank you for your interest in contributing to TEMPAD! This guide will help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/tempad.git`
3. Create a branch: `git checkout -b feat/your-feature`
4. Make your changes
5. Submit a pull request

## Development Setup

```bash
cd code/go

# Build
go build ./cmd/tempad

# Run tests
go test ./...

# Race detector
go test -race ./...

# Vet
go vet ./...
```

## Code Guidelines

- **Go 1.22+** features are welcome
- **No global state** — pass dependencies through function parameters or struct fields
- **Context propagation** — all I/O operations take `context.Context` as the first parameter
- **Typed errors** — use custom error types with `errors.Is`/`errors.As` support
- **Structured logging** — use `slog` with contextual fields
- **Table-driven tests** — preferred for pure functions

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

## Branch Naming

```
feat/short-description
fix/short-description
refactor/short-description
```

## Reporting Issues

- Use the GitHub issue templates
- Include Go version (`go version`)
- Include OS and architecture
- Provide steps to reproduce

## Architecture

Before making significant changes, please read:

- [Spec](docs/SPEC_v1.md) — what TEMPAD does
- [Architecture](code/go/docs/ARCHITECTURE_GO_v1.md) — how the Go implementation is structured

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
