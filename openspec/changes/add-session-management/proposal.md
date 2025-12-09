# Change: Session Management

## Why
Need named sessions for Claude Code integration. Currently no way to identify or track sessions across terminal tabs.

## What Changes
- New `cly claude` command that wraps Claude Code with session management
- `--name` flag for explicit session names (or auto-generate)
- Reads/exports `CLAUDE_SESSION_NAME` environment variable
- Zellij tab rename via `zellij action rename-tab`

## Impact
- New `modules/claude/` module
- Uses `pkg/session/` for session logic
- No changes to existing commands
