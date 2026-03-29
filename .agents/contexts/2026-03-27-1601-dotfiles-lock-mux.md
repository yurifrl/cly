---
created: 2026-03-27T16:01:00Z
project: cly
description: Add lock file to dotfiles module for tracking and auto-cleaning removed entries
context: modules/dotfiles — symlinks, jobs, jsonc copies, op mappings, install commands
tags: [dotfiles, lock, cleanup, interactive]
session_name: 2026-03-27-1601-dotfiles-lock-mux
purpose: Implement a lock file so cly dotfiles auto-cleans stale artifacts when entries are removed from dotfiles.conf
session_id: 676b1c93-f202-4e02-ad7e-f0930a52789f
provider: pi
resume_with: cly agent-session resume --provider pi 2026-03-27-1601-dotfiles-lock-mux
context_name: 2026-03-27-1601-dotfiles-lock-mux
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-03-27-1601-dotfiles-lock-mux.md
---

## Session

- **Name:** 2026-03-27-1601-dotfiles-lock-mux
- **Purpose:** When entries are removed from `dotfiles.conf`, cly should automatically clean up stale artifacts rather than leaving them behind silently.
- **Resume:** `cly agent-session resume --provider pi 2026-03-27-1601-dotfiles-lock-mux`

## Context

The `cly dotfiles` command applies a declarative `dotfiles.conf` — creating symlinks, copying jsonc→json, running install commands, registering launchd jobs, and injecting 1Password templates. Before this session there was no memory of previous runs, so removing an entry from the config left the old artifact (symlink, plist, generated file) silently lingering.

## Problem

No mechanism to detect and clean up stale dotfiles artifacts after config entries are removed. Every re-run would re-apply current state but never undo past state.

## Decisions

1. **Lock location:** `~/.local/share/cly/dotfiles/dotfiles.lock` — system data dir, consistent with `jobs-state.json`. Keeps dotfiles repo clean.
2. **Op mappings on removal:** auto-delete the generated destination file.
3. **Install commands on removal:** interactive pause requiring Enter. Cannot auto-undo, user must act manually. Bypassable with `--no-it`.
4. **Jobs/symlinks/jsonc on removal:** fully auto-clean, no prompt.

## Current State

**Complete and passing.** All 95 dotfiles tests pass. Build succeeds.

### Files created
- `modules/dotfiles/lock.go` — `DotfilesLock`, `LockDiff`, `lockFilePath`, `loadLock`, `saveLock`, `buildLock`, `diffLocks`
- `modules/dotfiles/lock_test.go` — 9 tests covering build, diff, round-trip, directory creation

### Files modified
- `modules/dotfiles/cmd.go` — `--no-it` flag; lock load/diff/save in `runSync`; `applyDiff()` function with interactive pause for removed install commands
- `modules/dotfiles/jobs.go` — `RemoveJobByName(name string)` helper
- `modules/dotfiles/jsonc.go` — `RemoveJsoncCopy(m Mapping) bool` helper
- `modules/dotfiles/op.go` — `RemoveOpMapping(m OpMapping) bool` helper

## Next Steps

- Commit the changes
- Could add `cly dotfiles lock` subcommand to inspect lock file contents (nice-to-have)
- Could persist which install commands ran successfully vs failed (to avoid warning about commands that errored out)
