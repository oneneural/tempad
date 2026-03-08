# T-P700: HTTP server setup and lifecycle

**Ticket:** T-P700 | **Phase:** 7 — HTTP Server | **Status:** 🔲 TODO
**Spec:** Section 15.5 | **Deps:** T-P501

## Description
Chi-based HTTP server binding loopback only with graceful shutdown.

## Files
- `internal/server/server.go`

## Work Items
- [ ] `NewServer(port, orchestrator)` with Chi router
- [ ] Bind `127.0.0.1:<port>` (loopback only)
- [ ] Graceful shutdown on context cancellation
- [ ] port=0 for ephemeral in tests
- [ ] 405 for unsupported methods, error envelope

## Acceptance Criteria
- [ ] Server starts, binds loopback
- [ ] Graceful shutdown works
- [ ] 405 for wrong methods

## Research Notes (2026-03-08)

- **Chi router**: `go-chi/chi/v5` confirmed as the right choice — lightweight, stdlib `net/http` compatible, excellent middleware ecosystem.
- Use `chi.NewRouter()` with `middleware.Logger`, `middleware.Recoverer`, and `middleware.Timeout(30 * time.Second)`.
- **Loopback**: `127.0.0.1` not `0.0.0.0` — prevents external access. This is a security requirement.
- **Graceful shutdown**: `http.Server{}.Shutdown(ctx)` with a 5-second deadline context.
- Consider registering `net/http/pprof` routes for debug builds behind a build tag.
