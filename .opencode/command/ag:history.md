---
description: Browse and search Claude Code conversation history
allowed-tools: Bash(fish:*), Read(**)
---

# History

Browse and search Claude Code conversation history.

## Instructions

1. **Read history index** from `~/.claude/history.jsonl`
   - Each line is a JSON object with session metadata
   - Parse: session ID, project path, timestamp, summary if available

2. **Display recent sessions** (last 20 by default):
```
Recent conversations:
[1] 2026-02-16 14:23 - DotFiles - "Updated fish completions"
[2] 2026-02-16 10:15 - myproject - "Fixed auth bug"
...
[q] Cancel
```

3. **Ask user** what they want:
   - Pick a session number to view details
   - Type a search term to filter by project or content
   - `q` to cancel

4. **When session selected**:
   - Find full conversation in `~/.claude/projects/` matching the project path
   - Read the session file
   - Display a summary: timestamps, key topics discussed, tools used, files modified
   - Ask if they want to resume with `/resume` or just view

5. **Search mode** (if user types text instead of number):
   - Filter sessions by project name or summary content
   - Re-display filtered list

## Notes

- History index: `~/.claude/history.jsonl`
- Full conversations: `~/.claude/projects/<project-path>/`
- This is read-only — never modify history files
