# Statusline

Claude Code status line integration. Renders context usage, model, cost, and custom commands for terminal status bars.

## Setup

Add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "command": "cly statusline"
  }
}
```

## Commands

```bash
cly statusline          # Full status line
cly statusline context  # Context window only
cly statusline model    # Model name only
cly statusline cost     # Session cost only
```

## Config

In `~/.config/cly/config.yaml`:

```yaml
statusline:
  format: "$context │ $model │ $cost"
  context:
    enabled: true
  model:
    enabled: true
  cost:
    enabled: true
  custom:
    enabled: false
    command: "echo hello"
    timeout: 500
```

## References

- [claude-code-statusline](https://github.com/rz1989s/claude-code-statusline)
- [cc-statusline](https://github.com/chongdashu/cc-statusline)
- [ccstatusline](https://github.com/sirmalloc/ccstatusline)
