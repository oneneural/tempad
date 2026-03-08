# T-P600: WORKFLOW.md file watcher with debounce

**Ticket:** T-P600 | **Phase:** 6 — Hot Reload + Logging | **Status:** 🔲 TODO
**Spec:** Section 8.2 | **Deps:** T-P105

## Description
fsnotify-based WORKFLOW.md watcher with 500ms debounce. On change: re-parse → re-merge → validate → send on channel if valid, log error if invalid.

## Files
- `internal/config/watcher.go`

## Work Items
- [ ] `StartWatcher(path, reload chan<-)` using fsnotify
- [ ] 500ms debounce on change events
- [ ] Valid → send new config. Invalid → log error, keep last known good.
- [ ] Handle rename-and-replace (re-add watch)

## Acceptance Criteria
- [ ] Edit WORKFLOW.md → new config within 1s
- [ ] Rapid edits → single reload
- [ ] Invalid edit → error logged, old config kept
- [ ] File deleted+recreated → re-watched

## Research Notes (2026-03-08)

- **CRITICAL**: Watch the **directory** containing WORKFLOW.md, not the file itself. Vim, Emacs, and many editors use rename-and-replace (write tmp → rename tmp to target), which causes fsnotify to lose the file watch.
- **Pattern**: `watcher.Add(filepath.Dir(workflowPath))` → filter events where `event.Name == workflowPath`.
- **Debounce**: Use `time.AfterFunc` with reset on each event — simpler than ticker-based debounce. Each new event calls `timer.Reset(500ms)`.
- The rename-and-replace handling becomes automatic when watching the directory — you'll get a `Create` event for the new file.
