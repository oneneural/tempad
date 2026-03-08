# T-P505: Agent worker goroutine

**Ticket:** T-P505 | **Phase:** 5 — Daemon Mode | **Status:** 🔲 TODO
**Spec:** Section 19.5, 11.3-5 | **Deps:** T-P302, T-P108, T-P506

## Description
runWorker: workspace.Prepare → prompt.Render → deliver → set 7 env vars → agent.Launch → Wait → after_run hook → WorkerResult. Tee stdout/stderr to log + stall tracking

## Files
- `internal/orchestrator/worker.go`

## Work Items
- [ ] Implement as described above
- [ ] Unit tests

## Acceptance Criteria
- [ ] Implementation matches spec
- [ ] Unit tests pass
- [ ] No race conditions (`go test -race`)

## Research Notes (2026-03-08)

- **7 environment variables**: `TEMPAD_ISSUE_ID`, `TEMPAD_ISSUE_IDENTIFIER`, `TEMPAD_ISSUE_TITLE`, `TEMPAD_ISSUE_URL`, `TEMPAD_WORKSPACE_DIR`, `TEMPAD_PROMPT_FILE`, `TEMPAD_LOG_FILE`.
- Use `slog.With("issue", issue.Identifier)` for per-worker structured logging context.
- Worker goroutine must defer sending `WorkerResult` even on panic (use `recover()`).
- Tee stdout/stderr using `io.TeeReader` to both log file and stall tracker simultaneously.
