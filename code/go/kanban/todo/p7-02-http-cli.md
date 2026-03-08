# T-P702: HTTP server CLI integration

**Ticket:** T-P702 | **Phase:** 7 — HTTP Server | **Status:** 🔲 TODO
**Spec:** Section 15.5, 18.1 | **Deps:** T-P701, T-P512

## Description

Wire --port flag to start HTTP server alongside daemon orchestrator.

## Files

- `cmd/tempad/main.go` (expand)

## Work Items

- [ ] --port flag: if > 0 AND daemon → start server
- [ ] TUI mode with --port → warn
- [ ] Log bound address on startup

## Acceptance Criteria

- [ ] `tempad --daemon --port 8080` starts both
- [ ] `curl localhost:8080/api/v1/state` returns JSON
- [ ] TUI + --port → warning

## Research Notes (2026-03-08)

- Use `net.Listen("tcp", addr)` first to check port availability, then pass listener to `http.Serve()`.
- Log actual bound address (important when port=0 for ephemeral).
