# Phase 3: Workspace Manager + Hooks

**Status:** 🔲 PENDING
**Tickets:** T-P300 to T-P304 (5 tickets)
**Prerequisites:** Phase 1 (complete). Phase 2 needed only for T-P304 (tracker-based clean).
**Goal:** Deterministic workspace creation, hook execution, safety invariants, cleanup.

## Success Criteria

- [ ] Deterministic workspace paths per issue identifier
- [ ] Path sanitization prevents traversal attacks
- [ ] `after_create` runs only on new workspace
- [ ] `before_run` failure aborts attempt
- [ ] Hook timeout kills process group
- [ ] Workspace cleanup works for terminal issues and manual
- [ ] `tempad clean` and `tempad clean <identifier>` work
- [ ] Path traversal attempts rejected (never removes paths outside root)

## Task List

| # | Ticket | Task | Status | File | Deps |
|---|--------|------|--------|------|------|
| 1 | T-P300 | Workspace path resolution and safety | 🔲 Todo | `p3-00-workspace-paths.md` | T-P101 |
| 2 | T-P301 | Hook execution engine | 🔲 Todo | `p3-01-hook-execution.md` | T-P100 |
| 3 | T-P302 | Workspace Prepare lifecycle | 🔲 Todo | `p3-02-workspace-prepare.md` | T-P300, T-P301 |
| 4 | T-P303 | Workspace cleanup (terminal + manual) | 🔲 Todo | `p3-03-workspace-cleanup.md` | T-P300, T-P301 |
| 5 | T-P304 | `tempad clean` CLI commands | 🔲 Todo | `p3-04-clean-cli.md` | T-P303, T-P204 |

## Dependency Order

```
{T-P300, T-P301} → T-P302 → T-P303 → T-P304
```

T-P300 and T-P301 can be done in parallel. T-P304 needs Phase 2's T-P204 for tracker-based cleanup.

## Parallelization with Phase 2

Phase 3 can be developed in parallel with Phase 2. Only T-P304 depends on Phase 2 — the rest are independent.

## Research Findings (2026-03-08)

**Key corrections:**
- **CRITICAL**: Path traversal prevention must use `filepath.Rel()` or `filepath.IsLocal()` (Go 1.20+), NOT `strings.HasPrefix()`.
- Process group kill must use negative PID: `syscall.Kill(-pid, sig)` to kill entire process group.

**Recommendations:**
- Hook timeout: use `context.WithTimeout` + goroutine with SIGTERM → SIGKILL escalation.
- Set `TEMPAD_ISSUE_ID`, `TEMPAD_WORKSPACE_DIR` env vars for hooks.
- Log hook stdout/stderr via `slog` at debug level.

See `handoffs/research-findings.md` for full details.
