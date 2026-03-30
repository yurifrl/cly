---
created: 2026-03-30T18:00:00Z
project: cly
description: Fix gc (git-commits) path doubling bug and add continue-on-error behavior
context: git-commits module; running gc from a repo subdirectory
tags: [git-commits, bugfix, executor]
session_name: 2026-03-30-1527-pi-tree-tui-tests
purpose: Fix gc failing with doubled path prefix when run from a git subdirectory, and make it continue committing remaining groups instead of rolling back on first failure.
session_id: 1b0f6310-6e22-4b78-b38e-ef6b01d8d629
provider: pi
resume_with: cly agent-session resume --provider pi 2026-03-30-1527-pi-tree-tui-tests
context_name: 2026-03-30-1527-pi-tree-tui-tests
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-03-30-1527-pi-tree-tui-tests.md
---

## Session

- **Name:** 2026-03-30-1527-pi-tree-tui-tests
- **Purpose:** Fix `gc` (`cly git-commits`) failing with a doubled path prefix when run from a subdirectory of a git repo, and change failure behavior to skip failed commits and continue.
- **Resume:** `cly agent-session resume --provider pi 2026-03-30-1527-pi-tree-tui-tests`

## Context

The user ran `gc` from `oncall/projects/bmm-certification/` (a subdirectory of the repo root). Git diff output returns paths relative to the repo root (e.g., `oncall/projects/bmm-certification/.bmm/sp-cache.json`), but all `exec.Command("git", ...)` calls ran without setting `cmd.Dir`, so git executed from the subdirectory and double-prefixed the path.

## Problem

1. **Path doubling:** `git add "oncall/projects/bmm-certification/.bmm/sp-cache.json"` run from `oncall/projects/bmm-certification/` caused git to look for `oncall/projects/bmm-certification/oncall/projects/bmm-certification/.bmm/` — fatal error, first commit fails, everything rolls back.
2. **Hard stop on first error:** On any commit failure the entire operation rolled back, leaving nothing committed.
3. **Noisy usage block:** Cobra printed the full usage text on any error return.

## Decisions

- Add `repoRoot()` helper in `changeset.go` that calls `git rev-parse --show-toplevel` to find the repo root.
- Set `cmd.Dir = repoRoot()` on every git exec: `gitExec`, `gitOutput`, `gitRawOutput`, and the `git apply` call in `rollback`.
- Change `Execute()` in `executor.go` to skip failed groups (unstage partial staging, record `Skipped: true` with the error) and continue rather than rolling back.
- `CommitResult` gains `Skipped bool` and `Err error` fields.
- `pipeline.go`: don't return error before printing results; show `✗ SKIP` for failed groups and `✓ <sha>` for successful ones; summary line shows `Done! Created N/M commits.`
- Add `SilenceUsage: true` to the cobra command so usage block is suppressed on errors.

## Current State

All changes built and verified with `go build ./modules/git-commits/...` and `task build`. No tests run (no test environment available for git integration tests in this session).

## Next Steps

- Run integration tests: `go test ./modules/git-commits/...`
- Test manually by running `gc` from a subdirectory of a real repo
- Consider caching `repoRoot()` result to avoid repeated subprocess calls per git operation
