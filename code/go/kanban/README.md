# Kanban Task Management System

This directory uses a simple file-based kanban system to track development tasks for T.E.M.P.A.D.

## Directory Structure

```plaintext
kanban/
├── README.md                    # This file - how the system works
├── plans/                       # Master plans and phase overviews
│   ├── phase-1-foundation.md
│   ├── phase-2-tracker.md
│   ├── phase-3-workspace.md
│   ├── phase-4-tui.md
│   ├── phase-5-daemon.md
│   ├── phase-6-polish.md
│   ├── phase-7-http.md
│   └── phase-8-hardening.md
├── todo/                        # Tasks waiting to be started
├── in-progress/                 # Tasks currently being worked on
├── done/                        # Completed tasks
├── parked/                      # Tasks deferred to later (not needed now)
└── handoffs/                    # Context docs for handing work between sessions/agents
```

## How It Works

This kanban system uses **git** to move tasks through four states:

1. **todo/** - Tasks waiting to be started
2. **in-progress/** - Tasks actively being worked on (ideally 1 at a time)
3. **done/** - Completed tasks
4. **parked/** - Tasks deferred to a later phase (not needed for current work)

Each task is a markdown file with detailed instructions, acceptance criteria, and checklists.

## Workflow

### Starting a Task

```bash
# Move task from todo to in-progress
git mv kanban/todo/p2-00-tracker-interface.md kanban/in-progress/

# Open the task file and follow instructions
open kanban/in-progress/p2-00-tracker-interface.md
```

### Implementing a Task

```bash
# 1. Read the task file for problem description, solution, and acceptance criteria
# 2. Make the code changes
# 3. Verify — run all checks

go build ./cmd/tempad
go test -race ./...
go vet ./...
```

### Completing a Task

```bash
# Move task from in-progress to done
git mv kanban/in-progress/p2-00-tracker-interface.md kanban/done/

# Stage code changes + kanban move, commit together
git add <changed-files> kanban/done/p2-00-tracker-interface.md kanban/in-progress/
git commit -m "feat(tracker): implement tracker client interface and error types

Implements T-P200. Defines Client interface with 6 operations and
all typed error categories per Spec Section 13.1, 13.4."
```

### Parking a Task

Defer tasks not needed for current phase. Update master plan with reason.

```bash
git mv kanban/todo/p7-00-http-server.md kanban/parked/
git commit -m "chore(kanban): park http-server (not needed until Phase 7)"
```

### Working on Multiple Tasks

While you can have multiple tasks in progress, it's recommended to:

- Keep 1-2 tasks in `in-progress/` at most
- Finish one before starting another when possible
- Use separate branches for independent work streams

## Task File Naming Convention

Tasks follow a phase-based naming pattern that maps to the product backlog:

```plaintext
# Phase-based (primary pattern for TEMPAD)
pX-YY-short-description.md

Where:
  X  = phase number (1-8, maps to PRODUCT_BACKLOG_v1.md phases)
  YY = task number within the phase (00-13, maps to backlog ticket IDs)

Examples:
- p1-00-go-module-scaffold.md         → Backlog ticket T-P100
- p1-01-domain-model-structs.md       → Backlog ticket T-P101
- p2-00-tracker-interface.md          → Backlog ticket T-P200
- p5-01-orchestrator-select-loop.md   → Backlog ticket T-P501

# Mapping: pX-YY → T-PXYY
# p1-00 → T-P100, p2-04 → T-P204, p5-12 → T-P512

# Topic-based (for bug fixes or work outside the backlog)
<topic>-<NN>-short-description.md

Examples:
- ci-01-github-actions-pipeline.md
- fix-01-config-loader-env-resolution.md
```

## Master Plans

Each phase has a master plan document in `kanban/plans/` that:

- Lists all tasks with priority and status
- Defines goals and success criteria
- Shows task dependencies and recommended order
- Tracks overall progress

Master plans are **living documents** — update them as tasks are completed.

## Handoffs

The `handoffs/` folder stores context documents for passing work between sessions or agents. Use these when:

- A task spans multiple work sessions
- An AI agent needs context from a previous session
- Complex state needs to be preserved between work handoffs

## Commit Message Convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```plaintext
<type>(<scope>): <description>

Types:
  fix      - Bug fixes
  feat     - New features
  refactor - Code restructuring without behavior change
  chore    - Maintenance tasks (kanban moves, config, etc.)
  test     - Adding or updating tests
  docs     - Documentation changes
  ci       - CI/CD pipeline changes

Scope (maps to TEMPAD's Go package structure):
  domain     - internal/domain (core domain model)
  config     - internal/config (config layer)
  tracker    - internal/tracker (tracker client)
  linear     - internal/tracker/linear (Linear implementation)
  workspace  - internal/workspace (workspace manager)
  prompt     - internal/prompt (prompt builder)
  agent      - internal/agent (agent launcher)
  claim      - internal/claim (claim mechanism)
  orchestrator - internal/orchestrator (daemon orchestrator)
  tui        - internal/tui (TUI mode)
  server     - internal/server (HTTP server)
  logging    - internal/logging (structured logging)
  cli        - cmd/tempad (CLI commands)
  kanban     - Kanban board changes only
  deps       - Dependency updates

Examples:
  feat(domain): define Issue, Workspace, and OrchestratorState structs
  feat(config): implement ServiceConfig merge with 5-level precedence
  feat(tracker): add Linear GraphQL client with pagination
  feat(tui): implement task board with Bubble Tea
  feat(orchestrator): add select loop with worker dispatch
  fix(config): handle empty $VAR resolution as missing
  test(orchestrator): add goroutine leak detection with goleak
  chore(kanban): complete p1-00 go module scaffold
```

## Branch & PR Workflow

Each task gets its own branch and PR — small, incremental, and reviewable.

### Branch Naming Convention

```plaintext
feat/p<X>-<YY>-<short-description>

Examples:
- feat/p2-00-tracker-interface
- feat/p2-01-graphql-queries
- feat/p3-00-workspace-paths
- feat/p5-01-orchestrator-loop
- fix/config-env-resolution
```

### Step-by-Step Process

```bash
# 1. Create task branch from the previous task's branch (or main for first in phase)
git checkout -b feat/p2-00-tracker-interface

# 2. Move task to in-progress
git mv kanban/todo/p2-00-tracker-interface.md kanban/in-progress/

# 3. Implement the task
#    - Make code changes
#    - Run verification: go build ./cmd/tempad && go test -race ./...
#    - Commit in multiple logical chunks

# 4. Move task to done
git mv kanban/in-progress/p2-00-tracker-interface.md kanban/done/
git add kanban/ && git commit -m "chore(kanban): complete p2-00 tracker interface"

# 5. Push and create PR
git push -u origin feat/p2-00-tracker-interface
gh pr create --title "feat(tracker): T-P200 implement tracker client interface" \
  --base main --body "..."
```

## Tracking Progress

### Count Tasks

```bash
# Total overview
echo "Todo: $(ls kanban/todo/*.md 2>/dev/null | wc -l) | In Progress: $(ls kanban/in-progress/*.md 2>/dev/null | wc -l) | Done: $(ls kanban/done/*.md 2>/dev/null | wc -l) | Parked: $(ls kanban/parked/*.md 2>/dev/null | wc -l)"
```

### Git History

```bash
# See task completion history
git log --oneline -- kanban/done/

# See when a specific task was completed
git log --all --full-history -- "kanban/done/p2-00-*.md"
```

## What Goes in Each Folder

### plans/
- Master plan documents for each phase
- Overview of all tasks, priorities, and dependencies

### todo/
- Tasks ready to be started
- Should have all prerequisites complete
- Ordered by dependency and priority

### in-progress/
- Tasks actively being worked on
- Ideally 1-2 tasks maximum
- Your current focus

### done/
- Completed tasks with all acceptance criteria met
- Serves as completion history
- Reference for similar future tasks

### parked/
- Tasks deferred to a later phase
- Not needed for current development work
- Should be documented in master plan with reason and revisit criteria

### handoffs/
- Context documents for session/agent handoffs
- Preserves state and decisions across work boundaries

## Best Practices

### Before Starting
- Read the master plan first
- Check task dependencies
- Understand acceptance criteria
- Review the relevant spec and architecture sections (listed in each task)

### While Working
- Follow step-by-step instructions in task file
- Check off items as you complete them
- Run `go build ./cmd/tempad && go test -race ./...` after each change
- Document any issues or deviations

### When Completing
- Verify all acceptance criteria met
- All Go checks pass (build, test, vet)
- Update master plan progress
- Commit with conventional commit message

---

**Note**: This is a living system. Adapt the workflow to what works best, but keep the git-based state tracking for visibility and history.

**Reference docs**: `../../docs/SPEC_v1.md` (behavior), `docs/ARCHITECTURE_GO_v1.md` (code structure), `docs/PRODUCT_BACKLOG_v1.md` (execution order), `../../docs/STACK_COMPARISON_v1.md` (tech stack decision).
