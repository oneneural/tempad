# T-P108: Prompt builder with Liquid templates

**Ticket:** T-P108
**Phase:** 1 — Foundation
**Status:** ✅ DONE
**Spec:** Section 6.4, Section 14

## Description

Liquid template rendering with strict variable checking via osteele/liquid.

## Files Created

- `internal/prompt/builder.go` — Builder, Render, issueToMap, DefaultPrompt
- `internal/prompt/builder_test.go` — 11 tests

## Acceptance Criteria

- [x] Renders templates with all issue fields
- [x] Unknown variable → template_render_error
- [x] attempt nil → {% if attempt %} skipped
- [x] Labels iterable in {% for %}
- [x] default filter works
- [x] Empty template → minimal default
- [x] 11 unit tests pass

## Dependencies

T-P101
