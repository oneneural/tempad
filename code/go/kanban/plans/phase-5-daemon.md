# Phase 5: Daemon Mode Orchestrator

**Status:** 🔲 PENDING
**Tickets:** T-P500 to T-P513 (14 tickets)
**Prerequisites:** Phase 2 (tracker client), Phase 3 (workspace manager)
**Goal:** `tempad --daemon` runs fully autonomous: poll → claim → dispatch → monitor → retry → reconcile.

## Success Criteria

- [ ] `tempad --daemon` starts orchestrator and auto-dispatches issues
- [ ] Concurrency limits respected (global + per-state)
- [ ] Agent exit code 0 → continuation retry (1s, doesn't count toward max)
- [ ] Agent exit code != 0 → exponential backoff retry
- [ ] Max retries exhausted → claim released
- [ ] Stall detection terminates stalled agents
- [ ] Reconciliation stops agents for terminal issues + cleans workspace
- [ ] All 7 agent env vars set (TEMPAD_ISSUE_ID, etc.)
- [ ] All 4 prompt delivery methods work (file, stdin, arg, env)
- [ ] Graceful shutdown on SIGINT/SIGTERM releases all claims
- [ ] No race conditions or goroutine leaks

## Task List

| # | Ticket | Task | Status | File | Deps |
| --- | -------- | ------ | -------- | ------ | ------ |
| 1 | T-P500 | Orchestrator runtime state | 🔲 Todo | `p5-00-orchestrator-state.md` | T-P101, T-P200, T-P300, T-P400 |
| 2 | T-P501 | Orchestrator main select loop | 🔲 Todo | `p5-01-select-loop.md` | T-P500 |
| 3 | T-P502 | Candidate selection and sorting | 🔲 Todo | `p5-02-candidate-selection.md` | T-P101 |
| 4 | T-P503 | Concurrency control | 🔲 Todo | `p5-03-concurrency-control.md` | T-P502 |
| 5 | T-P504 | Dispatch loop (claim + spawn) | 🔲 Todo | `p5-04-dispatch-loop.md` | T-P503, T-P400 |
| 6 | T-P505 | Agent worker goroutine | 🔲 Todo | `p5-05-worker-goroutine.md` | T-P302, T-P108, T-P506 |
| 7 | T-P506 | Prompt delivery (4 methods) | 🔲 Todo | `p5-06-prompt-delivery.md` | T-P100 |
| 8 | T-P507 | Agent subprocess launcher | 🔲 Todo | `p5-07-agent-launcher.md` | T-P506 |
| 9 | T-P508 | Agent output + stall detection | 🔲 Todo | `p5-08-output-stall.md` | T-P507 |
| 10 | T-P509 | Worker exit handling | 🔲 Todo | `p5-09-worker-exit.md` | T-P501, T-P510 |
| 11 | T-P510 | Retry scheduling and backoff | 🔲 Todo | `p5-10-retry-backoff.md` | T-P501 |
| 12 | T-P511 | Active run reconciliation | 🔲 Todo | `p5-11-reconciliation.md` | T-P508, T-P303 |
| 13 | T-P512 | Daemon mode entry point | 🔲 Todo | `p5-12-daemon-entrypoint.md` | T-P501–T-P511 |
| 14 | T-P513 | Daemon integration test | 🔲 Todo | `p5-13-daemon-integration.md` | T-P512 |

## Dependency Order

```text
T-P500 → T-P501 → {T-P502, T-P506, T-P507} → {T-P503, T-P508} → T-P504 → T-P505 → {T-P509, T-P510, T-P511} → T-P512 → T-P513
```

Several groups can be parallelized within the phase.

## Parallelization with Phase 4

Phase 5 (Daemon) and Phase 4 (TUI) are independent — they can be developed in parallel once Phases 2 and 3 are complete.

## Research Findings (2026-03-08)

**Key corrections:**

- **Channel buffer sizes**: Must use `make(chan WorkerResult, maxConcurrent)` to prevent goroutine leaks during shutdown.
- **Retry timers**: `time.AfterFunc` callbacks must check `ctx.Err() != nil` before modifying state.
- **Subprocess management**: `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pid, sig)` for process group kill.

**Recommendations:**

- Use `signal.NotifyContext` for clean signal handling.
- Use `cenkalti/backoff/v4` for exponential backoff with jitter.
- Use `lumberjack` for log rotation (slog has none built-in).
- Use `errgroup` for worker goroutine lifecycle management.
- Use `goleak` (uber-go/goleak) in all tests.

See `handoffs/research-findings.md` for full details.
