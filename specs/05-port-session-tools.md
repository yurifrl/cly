# Port Session Management Tools

**Goal**: Port session management features from claudext and claude-sessions to CLY

**Sources**:
- `~/DotFiles/home/.config/fish/functions/claudext.fish`
- `~/DotFiles/home/.local/bin/claude-sessions`

---

## Feature 1: Session Naming (from claudext)

### What it does
When running `claudext --name [NAME]` or `claudext` (no args):

1. Parse `--name` flag (optional value)
2. If `--name VALUE`: use VALUE as session name
3. If `--name` without value OR no flag: generate random ColorAnimal
4. Set env var: `CLAUDE_SESSION_NAME={name}`
5. Display: "🏷️ Session: {name}"
6. Pass args to Claude CLI

### Random Name Generator
- **Pattern**: ColorAnimal (RedWolf, BlueFox, etc.)
- **Colors** (14): red, blue, green, yellow, purple, orange, pink, cyan, magenta, lime, teal, navy, maroon, olive
- **Animals** (16): cat, dog, fox, wolf, bear, lion, tiger, shark, eagle, hawk, dove, owl, rabbit, deer, mouse, rat
- **Combinations**: 224 possible names

---

## Feature 2: Session Search & Resume (from claude-sessions)

### What it does
Search Claude Code session history and resume/fork sessions:

**Commands:**
- `claude-sessions` - List 20 most recent sessions
- `claude-sessions <query>` - Search by keyword in prompts
- `claude-sessions <session-id>` - Show session details
- `claude-sessions <id> -r` - Resume session in its project
- `claude-sessions <id> --fork-session` - Fork session
- `claude-sessions --since "2 days ago"` - Filter by date
- `claude-sessions --project DotFiles` - Filter by project

### Session Information Displayed
- Session ID (short: first 8 chars)
- Project path
- Git branch
- Start time (relative: "2h ago", "yesterday")
- Duration
- Message counts (user/assistant/thinking)
- File size
- Estimated tokens
- First/last message preview

### Resume Behavior
- Finds session in `~/.claude/projects/{project}/{session-id}.jsonl`
- Reads project path from `~/.claude/history.jsonl`
- Changes to project directory
- Executes: `claude --resume {session-id}`

---

## CLY Implementation

### Module: `modules/session/`

**Commands:**
```bash
# From claudext
cly session name [NAME]          # Generate/use name, set CLAUDE_SESSION_NAME

# From claude-sessions
cly session list [--limit N]     # List recent sessions
cly session search <query>       # Search sessions
cly session show <id>            # Show session details
cly session resume <id>          # Resume in project dir
cly session fork <id>            # Fork session
```

---

## Requirements

### Session Naming
- Generate random ColorAnimal if no name provided
- Accept user-provided name
- Set `CLAUDE_SESSION_NAME` environment variable
- Output name to stdout

### Session List
- Read from `~/.claude/history.jsonl`
- Show: session ID (short), age, project, first prompt
- Default limit: 20
- Sort by timestamp (newest first)

### Session Search
- Search in history.jsonl by keyword
- Match against display/prompt text
- Return matching sessions with metadata

### Session Details
- Read session file: `~/.claude/projects/{project}/{id}.jsonl`
- Parse JSONL entries
- Count messages (user/assistant/thinking)
- Calculate duration (first to last timestamp)
- Show project path, git branch
- Display first/last message preview

### Session Resume
- Look up project directory from history
- Verify project path exists
- Change to project directory
- Execute: `claude --resume {session-id}`

### Partial ID Resolution
- Accept partial session IDs (first 8+ chars)
- Search history.jsonl for matching full UUID
- Resolve to full session ID

---

## Data Sources

### `~/.claude/history.jsonl`
Each line is JSON with:
```json
{
  "sessionId": "uuid",
  "project": "/path/to/project",
  "timestamp": 1234567890000,
  "display": "first user prompt"
}
```

### `~/.claude/projects/{project}/{session-id}.jsonl`
Each line is JSON with:
```json
{
  "type": "user" | "assistant",
  "message": "content",
  "timestamp": 1234567890000,
  "gitBranch": "main",
  "thinkingMetadata": {...}
}
```

---

## Out of Scope

**Not porting:**
- Workspace navigation
- Folder creation
- Archive functionality
- Timestamp generation
- Interactive mode (TUI selection)
- Export to .paid (for now)
- Copy to clipboard

**Just**: Name generation + session search/resume from Claude Code history

---

## Module Location

`modules/session/` (utility module)

Files:
- cmd.go
- name.go (ColorAnimal generator)
- list.go (read history.jsonl)
- search.go
- show.go (metadata display)
- resume.go (cd + exec claude)
