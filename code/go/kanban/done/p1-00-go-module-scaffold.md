# T-P100: Initialize Go module and project scaffold

**Ticket:** T-P100
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 18.1

## Description

Initialize the Go module, create the full directory structure matching the architecture doc, stub `main.go` with Cobra root command, and add `.gitignore`.

## Files Created

- `code/go/go.mod` — Module `github.com/oneneural/tempad` with cobra, liquid, testify, yaml.v3
- `code/go/cmd/tempad/main.go` — Cobra root command with all flags
- `code/go/internal/` — Full directory tree (domain, config, tracker, workspace, prompt, agent, claim, orchestrator, tui, server, logging)
- `code/go/.gitignore` — Go binaries, IDE files, OS files
- Stub files for all future phase packages

## Acceptance Criteria

- [x] `go build ./cmd/tempad` compiles
- [x] `./tempad --help` prints usage
- [x] Directory structure matches Architecture doc Section 3

## Dependencies

None (start here)
