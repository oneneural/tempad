# T-P102: Workflow loader — parse WORKFLOW.md

**Ticket:** T-P102
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 6.1, 6.2, 6.3, 6.5

## Description

Implement `LoadWorkflow(path)` that parses YAML front matter from WORKFLOW.md, returning config map + prompt template string.

## Files Created

- `internal/config/workflow.go` — LoadWorkflow, parseWorkflow, splitFrontMatter, helpers
- `internal/config/workflow_test.go` — 6 test cases

## Acceptance Criteria

- [x] Loads WORKFLOW.md with front matter correctly
- [x] Missing file → `missing_workflow_file` error
- [x] Non-map YAML → `workflow_front_matter_not_a_map` error
- [x] No front matter → empty config, full file as prompt
- [x] Unknown keys ignored (forward compat)
- [x] 6 unit tests pass

## Dependencies

T-P100, T-P101
