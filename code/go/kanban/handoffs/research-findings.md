# Research Findings — Technical Validation

**Date:** 2026-03-08
**Scope:** All 8 phases validated against current best practices, API docs, blog posts, and community patterns.

## Summary

Four parallel research tracks were conducted to validate the technical approach in TEMPAD's backlog. Key corrections and recommendations are listed below, grouped by phase.

---

## Phase 2: Linear GraphQL API

### Validated

- **Auth**: Bearer token via `Authorization: Bearer <key>` header ✅
- **Pagination**: Cursor-based with `first`/`after` and `pageInfo { hasNextPage, endCursor }` ✅
- **Mutations**: `issueUpdate(id, input: { assigneeId })` for claim/unclaim ✅
- **Rate Limit**: 5,000 requests/hour per API key (complexity-weighted)

### Corrections

- **CRITICAL**: Use `project.slug` (not `slugId`) when filtering issues by project. The Linear API's `slugId` field is deprecated.
  - Fix in: T-P201, T-P202
- **GraphQL errors**: Linear returns HTTP 200 with `errors[]` array for GraphQL failures — must check response body even on 200.
  - Fix in: T-P202, T-P200 (error types)
- **Identity resolution**: Use `users(filter: { email: { eq: "..." } })` query to resolve email → user ID.
  - Fix in: T-P204

### Recommendations

- Use `@genqlient` or `shurcooL/graphql` for type-safe GraphQL client (avoid raw string queries)
- Add `X-Request-Id` header for debugging/tracing
- Cache user identity (email → ID) — it won't change during a session

---

## Phase 3: Workspace Manager + Hooks

### Validated

- **Hook execution**: `bash -lc` with `Setpgid: true` for process groups ✅
- **osteele/liquid**: Confirmed working with `StrictVariables` mode ✅
- **Timeout kill**: `syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)` for process group kill ✅

### Corrections

- **CRITICAL**: Path traversal prevention should use `filepath.Rel()` or `filepath.IsLocal()` (Go 1.20+), NOT just `strings.HasPrefix()`.
  - `filepath.Rel(root, candidate)` returns an error or starts with `..` if path escapes root — this is the canonical Go approach.
  - `filepath.IsLocal()` (Go 1.20+) checks if a path is local (no `..`, no absolute, no drive letters).
  - Fix in: T-P300
- **Process group kill**: Use negative PID: `syscall.Kill(-pid, sig)` to kill entire process group (not just lead process).
  - Fix in: T-P301

### Recommendations

- For hook timeout, use `context.WithTimeout` + goroutine that waits on `ctx.Done()` then sends SIGTERM → SIGKILL escalation
- Set `TEMPAD_ISSUE_ID`, `TEMPAD_WORKSPACE_DIR` env vars for hooks (useful for custom scripts)
- Log hook stdout/stderr via `slog` at debug level for troubleshooting

---

## Phase 4: Bubble Tea TUI

### Validated

- **Elm Architecture**: Model/Update/View confirmed as the right pattern ✅
- **tea.Tick**: For periodic polling (returns a Cmd that fires after duration) ✅
- **Selection preservation**: Store selected issue ID, re-find after poll refresh ✅

### Corrections

- **Polling deduplication**: Add a `pollInFlight bool` flag to prevent overlapping polls when API is slow.
  - In `Update`: only fire poll Cmd if `!m.pollInFlight`
  - Set flag on poll start, clear on `PollResultMsg`
  - Fix in: T-P405
- **Model composition**: Use embedded sub-models for each view (board view, detail view) with a `viewState` enum to switch between them — don't put everything in one flat struct.
  - Fix in: T-P401

### Recommendations

- Use `lipgloss` for consistent styling (colors, borders, padding)
- Use `teatest` for headless TUI testing (captures output, sends keystrokes)
- Consider `bubbles/list` component for the task board (built-in filtering, pagination)
- Implement `WindowSizeMsg` handler for responsive layout
- Use `BubbleUp` pattern: sub-models return commands that propagate status messages up to parent

---

## Phase 5: Daemon Mode (Orchestrator)

### Validated

- **Select loop pattern**: Standard Go `for-select` over ctx.Done/ticker/workerResults/retryTimers/configReload ✅
- **slog**: Correct choice for structured logging ✅
- **goleak**: uber-go/goleak for goroutine leak detection in tests ✅

### Corrections

- **Retry timers**: `time.AfterFunc` callbacks must check `ctx.Err() != nil` before modifying state — they can fire after shutdown has started.
  - Fix in: T-P510
- **Channel buffer sizes**: Use buffered channels for worker results (`make(chan WorkerResult, maxConcurrent)`) to prevent goroutine leaks when orchestrator is shutting down.
  - Fix in: T-P500, T-P501
- **Subprocess kill**: Must use `SysProcAttr{Setpgid: true}` on exec.Cmd AND kill with `syscall.Kill(-cmd.Process.Pid, sig)` for process groups.
  - Fix in: T-P507

### Recommendations

- Use `cenkalti/backoff/v4` library for exponential backoff with jitter (more robust than manual implementation)
- Use `lumberjack` (natefinch/lumberjack) for log file rotation (slog has no built-in rotation) — 50MB max, keep 5 files
- Add `slog.With("issue", id)` for per-issue structured logging context
- Graceful shutdown sequence: stop ticker → cancel workers → wait with timeout → release claims → exit
- Consider `errgroup` for managing worker goroutines (auto-cancels on first error if configured)

---

## Phase 6: Hot Reload + Logging

### Corrections

- **fsnotify**: Watch the **directory** containing WORKFLOW.md, not the file itself.
  - Vim, Emacs, and many editors use rename-and-replace (write tmp → rename tmp to target), which causes fsnotify to lose the watch.
  - Watch directory → filter for events matching WORKFLOW.md filename.
  - Fix in: T-P600
- **Log rotation**: slog has no built-in rotation — must use `lumberjack` as the `io.Writer`.
  - Fix in: T-P601

### Recommendations

- Use `fsnotify.NewWatcher()` → `watcher.Add(filepath.Dir(workflowPath))` → filter `event.Name == workflowPath`
- Debounce pattern: `time.AfterFunc` with reset on each event (simpler than ticker-based)
- Lumberjack config: `MaxSize: 50` (MB), `MaxBackups: 5`, `MaxAge: 30` (days), `Compress: true`

---

## Phase 7: HTTP Status Server

### Validated

- **Chi router**: `go-chi/chi/v5` confirmed as the right choice (lightweight, stdlib compatible) ✅
- **Loopback binding**: `127.0.0.1:PORT` for security ✅
- **Graceful shutdown**: `http.Server.Shutdown(ctx)` with deadline ✅

### Recommendations

- Use `chi.NewRouter()` with `middleware.Logger` and `middleware.Recoverer`
- JSON responses via `render.JSON(w, r, data)` from `go-chi/render`
- Add health endpoint: `GET /healthz` returning `{"status": "ok"}`
- Consider `net/http/pprof` registration for debug builds

---

## Phase 8: Testing + Hardening

### Validated

- **goleak**: `goleak.VerifyNone(t)` in TestMain or per-test ✅
- **Race detection**: `go test -race ./...` ✅

### Recommendations

- Use `testify/assert` and `testify/require` for cleaner test assertions
- Mock Linear API with `httptest.NewServer()` returning canned GraphQL responses
- Use `t.TempDir()` for workspace tests (auto-cleanup)
- For e2e test: mock tracker + mock agent subprocess, verify full claim→dispatch→complete cycle
- Add `go vet ./...` and `staticcheck ./...` to CI pipeline
- Consider `testcontainers-go` only if real Linear API tests need isolation

---

## Cross-Cutting Concerns

| Area | Finding | Impact |
| ------ | --------- | -------- |
| Linear `slug` vs `slugId` | Use `slug` — `slugId` is deprecated | Phase 2, 4, 5 |
| Path safety | `filepath.Rel()` over `HasPrefix` | Phase 3 |
| fsnotify | Watch directory, not file | Phase 6 |
| Log rotation | Need lumberjack (slog has none) | Phase 5, 6 |
| Process groups | `Setpgid` + negative PID kill | Phase 3, 5 |
| Retry safety | Check ctx before state mutation | Phase 5 |
| Channel buffers | Buffer = maxConcurrent | Phase 5 |
| Poll dedup | `pollInFlight` flag | Phase 4 |

---

**Next:** Apply these findings to individual task files and master plans.
