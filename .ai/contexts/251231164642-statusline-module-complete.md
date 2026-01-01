---
created: 2025-12-31T16:46:42
project: cly
description: Implemented statusline module for Claude Code context window display
context: OpenSpec change add-statusline-module, follows unix philosophy with composable subcommands
tags: [statusline, claude-code, tdd, complete]
---

# Statusline Module Implementation - Complete

## Context

Working on: `openspec/changes/add-statusline-module/`

Implemented statusline module that displays Claude Code session info (context window %, model, cost, custom commands) following Unix philosophy: small composable tools + config-driven convenience.

**Files created:**
- `modules/statusline/types.go` - StatusJSON, Config structs
- `modules/statusline/context.go` - Token calculation, color by threshold (green/yellow/red)
- `modules/statusline/cmd.go` - Main cmd + subcommands (context, model, cost) + format parsing + custom command with timeout
- `modules/statusline/context_test.go` - 8 tests
- `modules/statusline/cmd_test.go` - 9 tests
- `modules/statusline/integration_test.go` - 5 integration tests

**Config changes:**
- `pkg/config/config.go:60-71` - Added statusline default config (all disabled - BASIS pattern)
- `pkg/config/config.go:87-103` - Added StatuslineConfig types
- `pkg/config/config.go:245-283` - Added GetStatusline() method
- `~/.config/cly/config.yaml:57-68` - User config with context, model, starship enabled

**Registration:**
- `cmd/root.go:18` - Import statusline
- `cmd/root.go:59` - Register statusline.Register(RootCmd)

**Claude Code integration:**
- `~/.ai/ides/claude/settings.jsonc:368-372` - Already configured `"command": "cly statusline"`

## Problem

Claude Code pipes JSON with session data (model, workspace, transcript_path, cost, context_window) to statusline commands. Needed:
1. Display context window usage % with color coding
2. Unix-style composable tools (subcommands)
3. Config-driven main command for convenience
4. Custom command support (e.g., starship integration) with timeout protection
5. Format string to control output order

## Decisions

**BASIS pattern (all OFF by default):**
- Config has all components disabled by default
- User explicitly enables what they want
- Reasoning: User controls what shows, not opinionated defaults

**Unix philosophy - two interfaces:**
1. Subcommands (`cly statusline context`, `model`, `cost`) for manual shell composition
2. Main command (`cly statusline`) reads config, composes output
- Reasoning: Power users can pipe/wire themselves, convenience users get config-driven

**Custom command instead of starship-specific:**
- `custom.command` can run ANY command, starship is just an example
- Reasoning: More flexible, follows starship's own pattern (they support custom modules)

**Format string for order control:**
- `format: "$context │ $model │ $custom"` lets user control order
- Reasoning: Stolen from starship, proven UX pattern

**Timeout on custom commands:**
- Default 500ms, configurable
- Kills command if slow, returns empty
- Reasoning: Don't hang statusline on slow commands

## Current State

**Done:**
- ✅ All 22 tests passing
- ✅ TDD throughout (tests written first)
- ✅ Config integration with pkg/config
- ✅ Registered in cmd/root.go
- ✅ User config updated (~/.config/cly/config.yaml)
- ✅ Manual testing verified:
  - `cly statusline context` → `🧠 45% (90K/200K)`
  - `cly statusline model` → `[Opus]`
  - `cly statusline cost` → `💰 $0.05`
  - `cly statusline` with config → Full composed output with starship

**Not done (documented plugs for future):**
- Palettes/themes (catppuccin, nord, dracula)
- Presets (`cly statusline preset minimal`)
- Custom formats per component
- Substitutions

**Current config (~/.config/cly/config.yaml:57-68):**
```yaml
statusline:
  format: "$context │ $model │ $custom"
  context:
    enabled: true
  model:
    enabled: true
  cost:
    enabled: false
  custom:
    enabled: true
    command: "cd \"$cwd\" 2>/dev/null && starship prompt 2>/dev/null | sed 's/[❯>$]\\s*$//' | tr -d '\\n'"
    timeout: 500
```

## Next Steps

1. **User needs to restart Claude Code** to see statusline live (settings already configured)
2. **Future enhancements (if needed):**
   - Implement palette/theme support
   - Add preset configs
   - Custom symbol per component (currently hardcoded 🧠, 💰)
   - Separator config (currently hardcoded " │ ")
3. **Archive the OpenSpec change** after user confirms it works:
   ```bash
   openspec archive add-statusline-module
   ```

## Commands Reference

```bash
# Subcommands (Unix-style)
echo '{"context_window":...}' | cly statusline context
echo '{"model":{"display_name":"Opus"}}' | cly statusline model
echo '{"cost":{"total_cost_usd":0.05}}' | cly statusline cost

# Main (config-driven)
echo '{"model":...,"context_window":...,"workspace":...}' | cly statusline

# In Claude Code settings.jsonc
{
  "statusLine": {
    "command": "cly statusline",
    "padding": 0
  }
}
```

## Tests

All passing (22 total):
- Context: token calc, formatting, color thresholds
- Subcommands: context, model, cost rendering
- Format parsing: `$variable` extraction
- Config integration: enabled/disabled respect
- Custom command: execution, timeout protection
- Integration: full flow with real JSON input

Run: `go test ./modules/statusline/... -v`
