---
created: 2026-02-13T10:08:21
project: cly
description: Implemented cly agents module — daemon-based .agents config sync replacing Python ai script
context: Replaces ~/.local/bin/ai Python script. Patterns from pocs/file-syncer/. Plan was in openspec.
tags: [agents, sync, daemon, cli]
---

# cly agents — .agents config sync module

## Context
New module at `modules/agents/` (11 files). Syncs `~/.agents/` configs to IDE directories (`.claude/`, `.opencode/`, `.crush/`). Wired into CLI via `cmd/root.go`.

## Problem
The Python script `~/.local/bin/ai` did one-shot copying of `.agents/` to IDE dirs. Reimplemented as `cly agents` in Go with daemon mode for live-syncing via fsnotify.

## Decisions
- **Inline implementation** — all code in `modules/agents/`, not lifted to `pkg/`. Keeps it self-contained.
- **Source-only sync** — transforms are lossy (JSONC→JSON, SKILL.md stripping), so sync is source→target only.
- **JSONC parser hand-rolled** — no json5 dependency. Handles `//`, `/* */`, trailing commas, bare-word keys/values in ai.json.
- **Debounce in watcher** — two-timer pattern from POC (wait + maxWait).
- **Unix socket daemon** — JSON-over-newline protocol, same as POC `pocs/file-syncer/pkg/daemon/`.

## Current State
- Done: All 6 steps complete. 11 files created, 22 tests passing, full build clean.
- Done: `cmd/root.go` wired up with `agents.Register(RootCmd)`.
- Done: `cly agents sync --dry-run` works against real `~/.agents/`.
- Not committed: all changes are uncommitted.
- Pre-existing failures: `modules/mcp` and `modules/obsidian` tests were already broken, unrelated.

## Next Steps
1. Commit the new module
2. Test daemon mode manually (`cly agents daemon`, edit a file, verify sync)
3. Consider pruning support (removing target files when source is deleted)
4. Eventually retire the Python `~/.local/bin/ai` script
