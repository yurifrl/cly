# LLM Chat Module

Interactive AI chat powered by mods CLI tool - simple terminal interface.

## Features

- Simple stdin/stdout chat loop
- Conversation threading (compatible with mods)
- Styled output with colors
- Empty line to quit

## Usage

```bash
# Start new conversation
cly llm-chat

# Use different model
cly llm-chat --model claude-opus-4-5

# Continue existing conversation
cly llm-chat --continue modsi-1234567890
```

## Requirements

- `mods` binary must be in PATH
- `mods` configured with API key (via `mods --settings` or `~/Library/Application Support/mods/mods.yml` on macOS)

## Usage

Type your prompt and press Enter. Press Enter on an empty line to quit.

## Implementation

The module wraps the `mods` CLI tool as a subprocess, allowing seamless integration with mods' conversation storage and configuration. Uses mods' config file location (macOS: `~/Library/Application Support/mods/`, Linux: `~/.config/mods/`) and conversation cache.

### Architecture

- `client.go` - Wrapper around mods binary
- `cmd.go` - Cobra command registration + chat loop

### Testing

```bash
cd modules/llm-chat
go test -v
```

Integration tests require mods binary and ANTHROPIC_API_KEY.
