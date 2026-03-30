---
created: 2026-03-30T15:21:00-03:00
project: cly
description: Fix pi-tree enter key to open sessions in workspace
context: modules/pi-tree TUI keybindings
tags: [pi-tree, tui, bugfix]
session_name: pi-tree-enter-key-fix
purpose: Make enter key in pi-tree TUI open the selected session in an existing or new workspace
session_id: 49cd5a12-5947-4118-96f8-1d611b7af6fe
provider: pi
resume_with: cly agent-session resume --provider pi 2026-03-30-1521-pi-tree-tui-tests
context_name: 2026-03-30-1521-pi-tree-tui-tests
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-03-30-1521-pi-tree-tui-tests.md
---

## Session
- Name: pi-tree-enter-key-fix
- Purpose: Fix enter key in pi-tree TUI to open sessions
- Resume: `cly agent-session resume --provider pi 2026-03-30-1521-pi-tree-tui-tests`

## Problem
The enter key in `modules/pi-tree/tui.go` did nothing when pressed on a session row in the normal tree view. The `enter` case only handled the `showHist` branch and had no fallthrough logic for the main tree view. The open-session logic existed on the `d` key instead.

## Decisions
- Added the same open-session logic from the `d` key handler to the `enter` key handler (after the `showHist` block)
- Same behavior: checks if cwd matches session dir (execs `pi --session` locally) or calls `openSession()` to open in target workspace

## Current State
- Fix applied to `modules/pi-tree/tui.go`, compiles clean
- Not committed, not tested end-to-end

## Next Steps
- Test enter key in live pi-tree TUI
- Consider whether `d` key should still have the open logic or be repurposed (currently duplicated)
