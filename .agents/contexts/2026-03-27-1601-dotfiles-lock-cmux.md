---
created: 2026-03-27T16:01:00Z
project: cly
description: Add pkg/cmux integration package for cmux terminal multiplexer notifications
context: cmux notify integration, dotfiles module
tags: [cmux, notify, pkg, dotfiles]
session_name: 2026-03-27-1601-dotfiles-lock-cmux
purpose: Create a reusable pkg/cmux package so any module can send cmux notifications after finishing work, wired into dotfiles as the reference example.
session_id: c3c7de12-52fa-42d4-a8bf-e4134644345a
provider: pi
resume_with: cly agent-session resume --provider pi 2026-03-27-1601-dotfiles-lock-cmux
context_name: 2026-03-27-1601-dotfiles-lock-cmux
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-03-27-1601-dotfiles-lock-cmux.md
---

## Session

- **Name:** 2026-03-27-1601-dotfiles-lock-cmux
- **Purpose:** Create reusable `pkg/cmux` package for cmux notifications, wire into `modules/dotfiles`
- **Resume:** `cly agent-session resume --provider pi 2026-03-27-1601-dotfiles-lock-cmux`

## Context

User wanted cmux notification support that could be called from anywhere in the app (like `pkg/notify` is for Claude hooks), not a new CLI command. Started from a `notify send` command idea, pivoted to a shared package approach after clarifying the requirement.

Researched the cmux source repo (`manaflow-ai/cmux`) — found no first-party Go SDK. The `cmux` CLI binary itself is written in Go (`daemon/remote/cmd/cmuxd-remote/cli.go`) using stdlib `net` to dial a Unix socket. The CLI is the right integration surface; exec is correct.

## Problem

No reusable cmux integration existed. Each module would need to re-implement detection + exec boilerplate. The existing `pkg/notify` is coupled to Claude hook config and beeep/zellij — not appropriate for general use.

## Decisions

- **New package `pkg/cmux`** rather than extending `pkg/notify` — different concern (terminal multiplexer integration vs. Claude hook notifications)
- **exec-based** — matches how the zellij notifier works; the cmux CLI handles the socket protocol
- **`Available()` checks `CMUX_WORKSPACE_ID`** — documented env var set by cmux in all terminals
- **All functions no-op when unavailable** — callers don't need guards
- **Included `SetStatus`/`ClearStatus`** with `StatusOption` pattern for future use (e.g. progress during long operations)
- **Dotfiles as reference example** — `runSync` calls `cmux.Notify(cmd.Context(), "Dotfiles", "Sync complete")` at the end

## Current State

- `pkg/cmux/cmux.go` created — `Available()`, `Notify()`, `SetStatus()`, `ClearStatus()`
- `modules/dotfiles/cmd.go` updated — imports `pkg/cmux`, calls `Notify` at end of `runSync`
- Build passes clean (`task build`)
- Uncommitted — `pkg/cmux/` is untracked, `modules/dotfiles/cmd.go` is modified

## Next Steps

- Add `cmux.Notify` to other long-running commands (e.g. `bundle`, `install`, `backup`) following the same pattern
- Consider `cmux.SetStatus` for progress reporting during multi-step operations
- Commit and bump version
