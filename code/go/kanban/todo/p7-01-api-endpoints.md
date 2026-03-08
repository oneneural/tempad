# T-P701: API endpoints

**Ticket:** T-P701 | **Phase:** 7 — HTTP Server | **Status:** 🔲 TODO
**Spec:** Section 15.5 | **Deps:** T-P700

## Description

REST API and HTML dashboard endpoints for daemon observability.

## Files

- `internal/server/handlers.go`

## Work Items

- [ ] `GET /` — HTML dashboard
- [ ] `GET /api/v1/state` — JSON: running, retry queue, aggregates
- [ ] `GET /api/v1/<identifier>` — issue details, 404 if unknown
- [ ] `POST /api/v1/refresh` — trigger poll, 202 Accepted
- [ ] Thread-safe state snapshot

## Acceptance Criteria

- [ ] /api/v1/state returns correct JSON
- [ ] /api/v1/UNKNOWN → 404
- [ ] /api/v1/refresh triggers poll
- [ ] Dashboard renders

## Research Notes (2026-03-08)

- Use `go-chi/render` for JSON responses: `render.JSON(w, r, data)`.
- Add `GET /healthz` endpoint returning `{"status": "ok"}` for monitoring.
- Thread-safe state: orchestrator exposes `Snapshot() StateSnapshot` that copies state under a read lock.
- HTML dashboard via Go's `html/template` with `//go:embed templates/*.html`.
