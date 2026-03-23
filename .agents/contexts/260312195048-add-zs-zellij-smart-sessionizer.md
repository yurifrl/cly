---
created: 2026-03-12T19:51:16-03:00
project: cly
description: Added a new cly zs command that behaves like a smart Zellij sessionizer.
context: Follow-up implementation for a user request to copy the behavior of zellij-smart-sessionizer into CLY as a dedicated zs command.
tags: [zellij, zoxide, cobra, cli]
---

# Add `zs` Zellij smart sessionizer

## Context
Working in the CLY repo to add a dedicated `cly zs` command inspired by `zellij-smart-sessionizer`.

Relevant files:
- `modules/zs/cmd.go` — Cobra registration and inside/outside Zellij flow
- `modules/zs/picker.go` — fuzzy picker, dependency checks, HOME path shortening
- `modules/zs/session.go` — Zellij attach/create/tab actions and zoxide directory listing
- `modules/zs/layout.go` — layout discovery, filtering, preview, and selection
- `modules/zs/*_test.go` — focused tests for parsing and helper logic
- `cmd/root.go` — registers the new `zs` module

Reference used: `https://github.com/di-rs/zellij-smart-sessionizer/blob/master/zellij-smart-sessionizer`

## Problem
The user wanted a command like `zellij-smart-sessionizer` inside CLY: one interactive command that combines existing Zellij sessions and zoxide directories, attaches when a session already exists, creates a new session otherwise, and opens a new tab when already inside Zellij.

## Decisions
- Built a new self-contained module at `modules/zs/` instead of extending `modules/zl/` because project rules prefer isolated modules with no cross-module imports.
- Duplicated the minimal Zellij/zoxide exec helpers instead of extracting shared code to `pkg/` because the logic is small and this kept the change focused.
- Used `fzf`/`sk`-style shell pickers instead of Bubble Tea because the reference tool is shell-driven and the fastest path to matching behavior was to keep the picker external.
- Added layout selection with `--layout` and `--no-layout` flags so the command can be both interactive and scriptable.
- Kept changelog scope narrow to the `zs` feature only because the repo has many unrelated in-flight changes.

## Current State
- Done: added and registered `cly zs`
- Done: outside Zellij it shows sessions + zoxide dirs and attaches/creates appropriately
- Done: inside Zellij it opens a new tab for the selected zoxide directory
- Done: layout discovery and optional preview support are implemented
- Done: focused tests pass for `./modules/zs/...` and `./cmd/...`
- Done: `task build` succeeds
- Not done: no extra polish pass yet for richer labels, saved mappings, or UX refinements
- Blocked on: nothing for this feature; full `go test ./...` still has unrelated existing failures in `modules/mcp`

## Next Steps
1. Manually dogfood `cly zs` in a real Zellij environment to validate the picker and layout flows.
2. Optionally add nicer picker labels/previews or saved session→dir mappings if desired.
3. If wanted, update docs/help text or shell setup notes so `zs` is surfaced alongside `zl`.
