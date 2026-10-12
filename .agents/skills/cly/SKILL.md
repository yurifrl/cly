---
name: "cly"
description: "Personal CLI for session management, Zellij integration, and dev workflows"
allowed-tools:
  - Bash
---

# cly

Personal CLI at `/usr/local/bin/cly`. Source at `$WORKDIR/Yuri/cly`.

## Session Management

```bash
cly c -n <name>           # Start Claude with named session + rename Zellij tab
cly c -s <name>           # Resume or create a named session
cly c -a                  # Anonymous mode
cly c --yolo              # Skip permissions (safe-autonomous-mode prompt)
cly claude-rename <name>  # Rename Zellij tab from within a running session
```

`CLAUDE_SESSION_NAME` is set once at exec — only the tab can be renamed at runtime.

## Other Commands

```bash
cly zl <name>          # Zellij + zoxide: open pane in dir matching <name>
cly mcp                # Manage MCP servers
cly notify             # Send desktop notification
cly claude-session     # List/manage saved sessions
```
