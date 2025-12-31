# Statusline Module

New module at `modules/statusline/`

## Goal

Display context window % by parsing transcript JSONL.

## Input

Claude Code pipes JSON to stdin containing `transcript_path`.

## Output

Single line to stdout: `🧠 45% (90K/200K)`

## Files

- `modules/statusline/cmd.go` - Register(), read stdin, output stdout
- `modules/statusline/parser.go` - JSONL parsing, token calculation
- `modules/statusline/types.go` - Input, TranscriptEntry, Usage structs
- `modules/statusline/parser_test.go`

## Integration

- `cmd/root.go`: import and call `statusline.Register(RootCmd)`

## Logic

- Read `transcript_path` from stdin JSON
- Scan JSONL, find last `usage` entry
- Calculate: `input_tokens + cache_read_input_tokens + cache_creation_input_tokens`
- Percentage = total * 100 / 200000
- Format with `pkg/style` colors

## Output Format

```
🧠 45% (90K/200K)      # green, 0-50%
🧠 75% (150K/200K) ⚠️  # yellow, 50-75%
🧠 95% (190K/200K) 🔴  # red, 75%+
```

## User Config

`~/.claude/settings.json`:
```json
{
  "statusLine": {
    "type": "command",
    "command": "cly statusline",
    "padding": 0
  }
}
```
