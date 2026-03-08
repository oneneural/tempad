# TEMPAD Tech Stack Comparison & Recommendation

| | |
| --- | --- |
| **Version** | 1.0.0 |
| **Date** | 2026-03-07 |
| **Authors** | Subodh / Claude |
| **Status** | Decision — Go selected (score 116 vs Rust 101, Elixir 93) |

---

## 1. Evaluation Criteria

Every requirement below is extracted directly from the TEMPAD spec (docs/SPEC_v1.md). Each stack is scored on a 1–5 scale per requirement, where 5 = best-in-class and 1 = significant friction.

The criteria are weighted by importance to TEMPAD's success:

| Weight | Criteria | Why it matters |
| -------- | ---------- | --------------- |
| **Critical** | Subprocess management | Core daemon-mode contract: spawn, monitor, kill, timeout, capture stdout/stderr |
| **Critical** | TUI framework | Default operating mode; task board, live refresh, keyboard navigation |
| **Critical** | Distribution model | Developer tool must "just work" — download and run |
| **Critical** | Concurrency model | Daemon mode runs N agents + poll loop + reconciliation + retry timers simultaneously |
| **High** | File watching | Dynamic WORKFLOW.md reload without restart |
| **High** | Liquid template engine | Spec requires strict mode (unknown vars/filters must fail) |
| **High** | GraphQL client | Linear API with pagination, auth, error handling |
| **High** | YAML parsing | Config layer (WORKFLOW.md front matter + user config) |
| **Medium** | HTTP server | Optional extension (Section 15.5) |
| **Medium** | Structured logging | Key=value format, multiple sinks, issue context |
| **Medium** | CLI parsing | Commands, flags, subcommands |
| **Low** | Development velocity | Time-to-first-working-prototype |

---

## 2. Stack Profiles

### 2.1 Go

**Philosophy:** Simple, fast compilation, goroutines for concurrency, single static binary.

| Requirement | Library | Score | Notes |
| ------------ | --------- | ------- | ------- |
| Subprocess mgmt | `os/exec` (stdlib) | **5** | Battle-tested. `CommandContext` for timeout. Goroutines for pipe readers. |
| TUI framework | Bubble Tea (`charmbracelet/bubbletea`) | **5** | Elm Architecture. Production-proven (10K+ apps). Rich widget ecosystem (Lip Gloss for styling, Bubbles for components). |
| Distribution | `go build` + GOOS/GOARCH | **5** | Zero-setup cross-compilation. 5–15 MB static binary. No runtime deps. |
| Concurrency | Goroutines + channels + context | **5** | CSP model. Lightweight (2 KB/goroutine). Context cancellation propagates cleanly. Perfect for poll loop + N workers + retry timers. |
| File watching | `fsnotify/fsnotify` | **4** | Cross-platform. Needs manual debouncing. Rename-and-replace pattern needs re-watch. |
| Liquid templates | `osteele/liquid` | **5** | Pure Go Liquid implementation. **Has `StrictVariables` option** — unknown vars error. Matches spec requirement exactly. |
| GraphQL client | `hasura/go-graphql-client` | **4** | Solid client. Manual pagination. No Linear SDK (TypeScript only). |
| YAML parsing | `gopkg.in/yaml.v3` | **5** | Mature, pure Go. Struct tags for typed deserialization. |
| HTTP server | `net/http` + `go-chi/chi` | **5** | Stdlib is production-grade. Chi adds routing. Minimal deps. |
| Structured logging | `log/slog` (stdlib, Go 1.21+) | **5** | Official structured logging. Key=value native. Multiple handlers. |
| CLI parsing | `spf13/cobra` | **5** | Industry standard (Kubernetes, Hugo). Subcommands, flags, completions. |
| Dev velocity | — | **5** | Fast compilation. Simple language. Quick onboarding. |

**Go Total: 58/60**

**Go's Killer Advantages for TEMPAD:**

- Bubble Tea is arguably the best TUI framework in any language — purpose-built, mature, beautiful defaults
- Distribution is trivial: `GOOS=darwin GOARCH=arm64 go build -o tempad` and ship
- `os/exec` + goroutines is the exact concurrency model TEMPAD needs (poll loop in one goroutine, each agent worker in another, retry timers via `time.AfterFunc`)
- Go's `osteele/liquid` has **strict variable mode** — the only ecosystem where this works out of the box
- `slog` in stdlib means zero-dep structured logging
- Fast compile-test cycle accelerates development

**Go's Weaknesses:**

- No sum types / pattern matching — error handling is verbose
- No generics until Go 1.18 (now available but ecosystem still catching up)
- Goroutine leaks require discipline (always wire `context.Done()`)
- GC pauses are negligible for TEMPAD's workload but exist

---

### 2.2 Rust

**Philosophy:** Zero-cost abstractions, memory safety without GC, maximum performance, single binary.

| Requirement | Library | Score | Notes |
| ------------ | --------- | ------- | ------- |
| Subprocess mgmt | `tokio::process` | **5** | Async subprocess. Timeout via `tokio::time::timeout`. Solid. |
| TUI framework | Ratatui | **4** | Mature, performant. But requires manual event loop architecture. Higher learning curve than Bubble Tea. |
| Distribution | `cargo build --release` | **5** | Static binary via musl target. 5–20 MB. `cross` for cross-compilation. |
| Concurrency | Tokio async runtime | **4** | Powerful but lower-level. `'static` lifetime requirements complicate state sharing. Arc/Mutex patterns needed. |
| File watching | `notify` | **5** | 62M+ downloads. Cross-platform. Mature. |
| Liquid templates | `liquid` crate | **2** | **No strict mode.** Unknown variables silently resolve to empty. Custom wrapper needed to enforce TEMPAD's spec requirement. |
| GraphQL client | `graphql_client` | **4** | Compile-time type safety from schema. But schema must be fetched/cached. Iteration slower. |
| YAML parsing | `serde_yml` | **4** | Note: `serde_yaml` is archived (March 2024). Must use fork. Works well with Serde. |
| HTTP server | Axum | **5** | Tokio-native, modern, Tower middleware. Excellent. |
| Structured logging | `tracing` | **5** | Tokio team maintained. Spans, events, structured fields. Best-in-class. |
| CLI parsing | `clap` v4 | **5** | De facto standard. Derive macros. Shell completions. |
| Dev velocity | — | **2** | Steep learning curve. Slow compile times. Ownership/lifetime friction. |

**Rust Total: 50/60**

**Rust's Killer Advantages for TEMPAD:**

- Zero-cost abstractions mean the daemon mode orchestrator would be extremely efficient
- Compile-time safety catches concurrency bugs before runtime
- `tracing` is the gold standard for structured observability
- Single binary distribution matches Go
- Type system prevents entire classes of bugs (null safety, exhaustive matching)

**Rust's Weaknesses:**

- **Liquid strict mode is missing** — this is a spec requirement and would need a custom solution
- Development velocity is 2–3x slower than Go for a project of this complexity
- Tokio's `'static` lifetime requirements make the orchestrator state machine harder to express
- Ratatui requires more boilerplate than Bubble Tea for equivalent TUI features
- `serde_yaml` being dead is a minor red flag (fork `serde_yml` works but ecosystem fragmentation)
- Team onboarding is significantly harder

---

### 2.3 Elixir/OTP

**Philosophy:** Actor model, fault tolerance, "let it crash", supervision trees, BEAM VM.

| Requirement | Library | Score | Notes |
| ------------ | --------- | ------- | ------- |
| Subprocess mgmt | `Port.open/2` + GenServer | **4** | OTP process monitoring works. But killing OS subprocesses requires explicit port handling. No process groups. |
| TUI framework | Ratatouille | **3** | Functional but small community. Depends on ex_termbox (native bindings). Far less mature than Bubble Tea or Ratatui. |
| Distribution | Burrito (Mix release + Zig) | **2** | Works but: 80–150 MB binaries, first-run extraction delay, Zig build dependency, cross-compilation complexity. |
| Concurrency | OTP (GenServer, Supervisor) | **5** | **Best concurrency model** for this problem. Supervisor trees, process isolation, fault tolerance. Symphony already proved this works. |
| File watching | `file_system` hex package | **4** | Mature (Phoenix uses it). Native OS backends. |
| Liquid templates | Solid | **5** | **Has `strict_variables: true` and `strict_filters: true`.** Symphony uses this exact library. Perfect match. |
| GraphQL client | Neuron / raw HTTP | **3** | Neuron is lightweight but minimal. No pagination helpers. Manual JSON handling. |
| YAML parsing | `yaml_elixir` | **5** | Stable, based on pure-Erlang `yamerl`. Battle-tested. |
| HTTP server | Plug + Cowboy (or Phoenix) | **5** | Phoenix is overkill; Plug + Cowboy is perfect for the optional REST API. |
| Structured logging | Built-in `Logger` | **4** | Good metadata support. Needs custom formatter for JSON/structured output. |
| CLI parsing | `OptionParser` (stdlib) | **4** | Built-in, adequate. No subcommand support without extra work. |
| Dev velocity | — | **4** | Productive language. But BEAM ecosystem is niche; fewer developers know it. |

**Elixir Total: 48/60**

**Elixir's Killer Advantages for TEMPAD:**

- OTP supervision trees are the **ideal** concurrency model for daemon mode — each agent worker is a supervised process, the orchestrator is a GenServer, retry timers are `Process.send_after/3`
- Symphony already proved this architecture works in production
- Solid (Liquid engine) has native strict mode — spec requirement satisfied
- Hot code reload means WORKFLOW.md changes could be applied without even restarting
- "Let it crash" philosophy matches TEMPAD's failure model perfectly

**Elixir's Weaknesses:**

- **Distribution is the critical weakness.** Burrito produces 80–150 MB binaries with first-run extraction. Go/Rust produce 5–20 MB binaries that run instantly. For a developer tool that people download and run, this matters enormously.
- **Ratatouille is the weakest TUI option.** Small community, native dependency on ex_termbox, fewer widgets, less documentation. Bubble Tea and Ratatui are far ahead.
- Niche language — harder to attract contributors to an open-source project
- BEAM runtime overhead for a single-user local tool is unnecessary (BEAM shines at millions of concurrent connections; TEMPAD has ~5 concurrent agents)

---

## 3. Requirement-by-Requirement Winner

| Requirement | Winner | Runner-up | Notes |
| ------------ | -------- | ----------- | ------- |
| Subprocess mgmt | **Go = Rust** (tie) | Elixir | All three are solid; Go/Rust are slightly simpler |
| TUI framework | **Go** (Bubble Tea) | Rust (Ratatui) | Bubble Tea is purpose-built and delightful |
| Distribution | **Go = Rust** (tie) | Elixir (distant) | 5–15 MB vs 80–150 MB; instant vs extract |
| Concurrency | **Elixir** (OTP) | Go (goroutines) | OTP is the ideal model; Go is very close |
| File watching | **Rust** (notify) | Go/Elixir (tie) | All adequate; Rust's notify is most mature |
| Liquid templates | **Elixir** (Solid) = **Go** (osteele) | Rust (broken) | Both have strict mode; Rust doesn't |
| GraphQL client | **Go** | Rust | Hasura client is cleanest; Rust needs schema |
| YAML parsing | **Go = Elixir** (tie) | Rust | Go/Elixir have mature, stable parsers |
| HTTP server | **All tied** | — | Solved problem in every language |
| Structured logging | **Rust** (tracing) | Go (slog) | Tracing is best-in-class; slog is great too |
| CLI parsing | **Go** (Cobra) | Rust (Clap) | Both excellent; Cobra has subcommand ecosystem |
| Dev velocity | **Go** | Elixir | Go compiles fast, language is simple |

---

## 4. Decision Matrix Summary

| Stack | Critical (×3) | High (×2) | Medium (×1) | Low (×1) | Weighted Total |
| ------- | --------------- | ----------- | ------------- | ---------- | --------------- |
| **Go** | 20 × 3 = 60 | 18 × 2 = 36 | 15 × 1 = 15 | 5 × 1 = 5 | **116** |
| **Rust** | 18 × 3 = 54 | 15 × 2 = 30 | 15 × 1 = 15 | 2 × 1 = 2 | **101** |
| **Elixir** | 14 × 3 = 42 | 17 × 2 = 34 | 13 × 1 = 13 | 4 × 1 = 4 | **93** |

---

## 5. Recommendation: **Go**

### Why Go Wins

**1. Best TUI story.** Bubble Tea is the most mature, best-documented, and most delightful TUI framework available. TEMPAD's default mode is TUI — this matters enormously. The Charm ecosystem (Bubble Tea + Lip Gloss + Bubbles + Huh) gives you a complete toolkit for building beautiful terminal interfaces.

**2. Trivial distribution.** `go build` produces a static binary. Cross-compilation is one environment variable (`GOOS`/`GOARCH`). No runtime, no extraction, no Zig toolchain. For an open-source developer tool, this is table stakes.

**3. Goroutines map perfectly to TEMPAD's concurrency model.** The daemon mode orchestrator needs: one goroutine for the poll loop, one per agent worker, `time.AfterFunc` for retry timers, channels for worker exit notifications, `context.Context` for cancellation propagation. This is exactly what Go was designed for.

**4. Liquid strict mode works.** The `osteele/liquid` package has `StrictVariables` — unknown template variables return errors. This satisfies the spec requirement with zero custom code.

**5. Fastest path to a working prototype.** Go's simplicity, fast compilation, and rich standard library mean the first milestone (config + workflow loader + CLI skeleton) can ship in days, not weeks.

**6. Open-source contributor friendliness.** Go is one of the most popular languages for CLI tools and developer infrastructure (kubectl, Docker, Terraform, Hugo). The contributor pool is large.

### What Go Gives Up (and Why It's Okay)

**Concurrency model isn't OTP.** True — Go doesn't have supervision trees. But TEMPAD's concurrency needs are modest (5–20 concurrent agents, not millions of processes). Goroutines + channels + context cancellation handle this cleanly. If a goroutine panics, `recover()` in a deferred function catches it. This isn't as elegant as OTP's "let it crash" but it works for TEMPAD's scale.

**No compile-time memory safety.** True — Go has a GC and doesn't prevent data races at compile time. But TEMPAD isn't a high-performance systems tool. The GC overhead is negligible for a tool that spends 99% of its time waiting on I/O (tracker API, subprocess exit, file changes). Use the race detector (`go test -race`) during development.

**No pattern matching / sum types.** True — error handling is more verbose. But Go 1.22+ has `errors.Join`, and the simplicity of `if err != nil` is well-understood by every Go developer.

---

## 6. Proposed Go Architecture Preview

```text
tempad/
├── cmd/tempad/          # CLI entry point (Cobra)
│   └── main.go
├── internal/
│   ├── config/          # Config layer (WORKFLOW.md + user config merge)
│   ├── workflow/        # Workflow loader (YAML front matter + prompt body)
│   ├── tracker/         # Tracker adapter interface + Linear implementation
│   │   ├── tracker.go   # Interface: FetchCandidates, Assign, Unassign, etc.
│   │   └── linear/      # Linear GraphQL client
│   ├── workspace/       # Workspace manager (create, hooks, cleanup, safety)
│   ├── orchestrator/    # Daemon mode state machine (poll, dispatch, reconcile, retry)
│   ├── agent/           # Agent launcher (subprocess, prompt delivery, env vars)
│   ├── prompt/          # Prompt builder (Liquid rendering with strict vars)
│   ├── tui/             # TUI mode (Bubble Tea task board, selection, status)
│   ├── server/          # Optional HTTP server (Chi, REST API, dashboard)
│   └── logging/         # Structured logging setup (slog handlers, sinks)
├── WORKFLOW.md          # Example workflow file
├── go.mod
└── go.sum
```

### Key Architectural Decisions

**Orchestrator as a struct with a run loop (not an actor):**

```go
type Orchestrator struct {
    state    *RuntimeState
    tracker  tracker.Client
    workspace workspace.Manager
    agent    agent.Launcher
    // ...
}

func (o *Orchestrator) Run(ctx context.Context) error {
    ticker := time.NewTicker(o.state.PollInterval)
    for {
        select {
        case <-ctx.Done():
            return o.shutdown()
        case <-ticker.C:
            o.tick()
        case result := <-o.workerResults:
            o.handleWorkerExit(result)
        case <-o.retrySignals:
            o.handleRetry(...)
        }
    }
}
```

**Agent workers as goroutines with context:**
Each dispatched agent runs in its own goroutine. The goroutine owns subprocess lifecycle. Results sent back via channel. Context cancellation kills the subprocess.

**TUI as a Bubble Tea program:**
The TUI is a `tea.Program` that receives messages from the tracker (poll results) and user input (keyboard). The orchestrator logic for TUI mode is simpler — just claim → workspace → IDE open.

---

## 7. Risk Mitigations

| Risk | Mitigation |
| ------ | ----------- |
| Goroutine leaks | Wire `context.Context` through every goroutine. Use `goleak` in tests. |
| No supervision trees | Use `recover()` in deferred functions. Log panics. Restart failed workers via retry loop. |
| File watcher edge cases | Debounce with 500ms timer. Re-watch on rename events. |
| GraphQL pagination | Implement cursor-based pagination helper once in the Linear client. |
| Liquid edge cases | Validate template on workflow load (catch errors early). |

---

## 8. What If We Chose Rust Instead?

Rust would produce an equally correct, higher-performance binary. Choose Rust if:

- You want compile-time memory safety guarantees
- You plan to eventually handle 100+ concurrent agents (TEMPAD scale doesn't need this)
- You value `tracing`'s observability over `slog` (marginal difference)
- Development velocity isn't a concern

The Liquid strict mode gap is the most significant issue. You'd need to wrap the `liquid` crate with custom validation — doable but adds maintenance burden.

## 9. What If We Chose Elixir Instead?

Elixir would produce the most elegant daemon-mode implementation. Choose Elixir if:

- You're willing to accept 80–150 MB binary sizes
- TUI is not the primary mode (Ratatouille is the weakest link)
- You want to reuse Symphony's architecture directly
- The contributor pool is Elixir-experienced

The distribution story is the dealbreaker for an open-source developer tool.

---

## 10. Verdict

**Go is the recommended stack for TEMPAD.**

It offers the best balance of: TUI quality (Bubble Tea), distribution simplicity (single binary, trivial cross-compilation), concurrency adequacy (goroutines + channels), ecosystem completeness (every library TEMPAD needs is mature), and development velocity (fast compilation, simple language, large contributor pool).

Next step: Approve this recommendation, then proceed to detailed architecture design and phased implementation plan.
