# Change: Add statusline module for Claude Code

## Why
Claude Code pipes session JSON to statusline commands. Need composable tools + config-driven defaults.

## Philosophy
- Unix-style subcommands for manual composition
- Config-driven main command for convenience
- Leave plugs for visual TUI config later (like starship)

## Commands
```bash
cly statusline              # config-driven, outputs composed line
cly statusline context      # just context piece
cly statusline model        # just model piece
cly statusline cost         # just cost piece
```

## Config (options with params, all OFF by default)
```yaml
statusline:
  format: "$context │ $model │ $custom"  # order control
  context:
    enabled: false       # 🧠 45% (90K/200K)
    # symbol: "🧠"       # future plug
    # warning: 50        # future plug
    # danger: 75         # future plug
  model:
    enabled: false       # [Opus]
    # format: "[$name]"  # future plug
  cost:
    enabled: false       # 💰 $0.02
    # symbol: "💰"       # future plug
  custom:
    enabled: false       # run any command
    # command: "cd $cwd && starship prompt | tr -d '\n'"
    timeout: 500         # ms, protect against slow commands
```

**BASIS** = all disabled. User turns on what they want.

Example: enable starship via custom:
```yaml
statusline:
  custom:
    enabled: true
    command: "cd $cwd && starship prompt | tr -d '\n'"
```

## Output
`cly statusline` with context + model enabled:
```
🧠 45% (90K/200K) │ [Opus]
```

With all disabled (BASIS): empty output

## User Settings
```json
{
  "statusLine": {
    "command": "cly statusline"
  }
}
```

## Input (from Claude Code)
```json
{
  "transcript_path": "/path/to/transcript.jsonl",
  "model": { "display_name": "Opus" },
  "workspace": { "current_dir": "/cwd" },
  "context_window": {
    "context_window_size": 200000,
    "current_usage": { "input_tokens": 90000 }
  },
  "cost": { "total_cost_usd": 0.02 }
}
```

## Files
```
statusline/
├── cmd.go           # Main cmd + subcommands
├── types.go         # StatusJSON, Config structs
├── context.go       # Context calculation
├── context_test.go
├── cmd_test.go
└── integration_test.go  # JSON in → output check
```

## Plugs (future, documented)

**Palettes/themes** - pre-defined color schemes:
```yaml
statusline:
  palette: "catppuccin"  # or "nord", "dracula"
```

**Presets** - ready-made configs:
```bash
cly statusline preset minimal
cly statusline preset full
```

**Other:**
- Custom formats per component (`format: "[$name]"`)
- Substitutions

## Impact
- New module: `modules/statusline/`
- Config addition: `pkg/config/`
- Register in `cmd/root.go`
