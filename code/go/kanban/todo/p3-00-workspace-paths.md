# T-P300: Workspace path resolution and safety invariants

**Ticket:** T-P300
**Phase:** 3 — Workspace Manager + Hooks
**Status:** 🔲 TODO
**Spec:** Section 12.1, 12.2, 12.6
**Deps:** T-P101

## Problem

Need deterministic, safe workspace path resolution. Must prevent path traversal attacks.

## Solution

`NewManager(root)` with `resolvePath(identifier)` that sanitizes identifiers and enforces root containment.

## Files

- `internal/workspace/manager.go`

## Work Items

- [ ] `NewManager(workspaceRoot)` constructor
- [ ] `resolvePath(identifier)`: sanitize via `domain.SanitizeIdentifier()`, `filepath.Join(root, key)`, verify prefix containment
- [ ] `ensureDir(path)`: `os.MkdirAll(path, 0755)`, detect newly created vs existing
- [ ] Reject path traversal (`..`)
- [ ] Error if non-directory file exists at path

## Acceptance Criteria

- [ ] `ABC-123` → `<root>/ABC-123`
- [ ] `ABC/123` → `<root>/ABC_123` (sanitized)
- [ ] `../../etc/passwd` → sanitized, still under root
- [ ] Non-directory file at path → error
- [ ] 5 path traversal scenario tests

## Research Notes (2026-03-08)

- **CRITICAL**: Do NOT use `strings.HasPrefix()` for path containment checks — it fails on `../root-escape` patterns and symlink attacks.
- **Use `filepath.Rel(root, candidate)`**: Returns an error or path starting with `..` if the candidate escapes root. This is the canonical Go approach.
- **Alternative**: `filepath.IsLocal()` (Go 1.20+) checks if a path is local (no `..`, no absolute paths, no drive letters). Use on the sanitized identifier before joining with root.
- **Recommended pattern**: `resolved := filepath.Join(root, sanitized)` → `rel, err := filepath.Rel(root, resolved)` → reject if `err != nil` or `strings.HasPrefix(rel, "..")`.
- Consider also checking `filepath.EvalSymlinks(resolved)` after creation to catch symlink-based escapes.
- Use `t.TempDir()` in tests for auto-cleanup temp directories.
