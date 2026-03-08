# Phase 7: HTTP Server Extension

**Status:** 🔲 PENDING
**Tickets:** T-P700 to T-P702 (3 tickets)
**Prerequisites:** Phase 5 (Daemon orchestrator)
**Goal:** Optional `--port` enables REST API and dashboard for daemon mode observability.

## Success Criteria

- [ ] `tempad --daemon --port 8080` starts both orchestrator and HTTP server
- [ ] Server binds loopback only (127.0.0.1)
- [ ] `GET /` serves HTML dashboard
- [ ] `GET /api/v1/state` returns JSON (running sessions, retry queue, aggregates)
- [ ] `GET /api/v1/<identifier>` returns issue-specific details (404 if unknown)
- [ ] `POST /api/v1/refresh` triggers immediate poll, returns 202
- [ ] Graceful shutdown works
- [ ] TUI mode with `--port` → warning logged

## Task List

| # | Ticket | Task | Status | File | Deps |
|---|--------|------|--------|------|------|
| 1 | T-P700 | HTTP server setup and lifecycle | 🔲 Todo | `p7-00-http-server.md` | T-P501 |
| 2 | T-P701 | API endpoints | 🔲 Todo | `p7-01-api-endpoints.md` | T-P700 |
| 3 | T-P702 | HTTP server CLI integration | 🔲 Todo | `p7-02-http-cli.md` | T-P701, T-P512 |

## Dependency Order

```
T-P700 → T-P701 → T-P702
```

Sequential — each builds on the previous.

## Research Findings (2026-03-08)

**Validated:**
- Chi router (`go-chi/chi/v5`) confirmed as the right choice — lightweight, stdlib compatible.
- Loopback binding (`127.0.0.1`) and `http.Server.Shutdown()` for graceful shutdown.

**Recommendations:**
- Use `go-chi/render` for JSON responses.
- Add `GET /healthz` endpoint for monitoring.
- Consider `net/http/pprof` registration for debug builds.
- Use `net.Listen` then `http.Serve` pattern for port availability check.

See `handoffs/research-findings.md` for full details.
