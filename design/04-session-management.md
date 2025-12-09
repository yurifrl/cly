# Session Management

Named CLI sessions with persistence and resumption.

---

## What It Does

**Named sessions** - Give CLI sessions memorable names instead of random IDs

**Auto-persistence** - Sessions save to config automatically

**Resume anywhere** - Switch between sessions by name

**Terminal sync** - Tab/pane names update to match session

---

## Usage Examples

### Start named session
```bash
# Explicit name
cli --session-name WorkProject

# Auto-generated name
cli --session-name

# From environment
export SESSION_NAME=WorkProject
cli
```

**What happens:**
- Prints: `🏷️  Session: WorkProject`
- Saves: `~/.config/app/sessions.json` with `{"WorkProject": "uuid-123"}`
- Exports: `SESSION_ID=uuid-123`
- Updates terminal tab to: `WorkProject`

### Resume session
```bash
# By name
cli --resume WorkProject

# By ID
cli --resume uuid-123

# From file
cli --resume-from session-uuid-123.save
```

**What happens:**
- Looks up session ID from storage
- Restores session state
- Updates terminal tab name
- Exports session context

### List sessions
```bash
cli --list-sessions
# Shows: WorkProject (2h ago), Research (1d ago), TempWork (3d ago)
```

---

## Features

### Session Naming

**Three sources:**
1. `--session-name NAME` - explicit flag
2. `SESSION_NAME=NAME` - environment variable
3. `--session-name` (no value) - auto-generate

**Auto-generated format:**
- Pattern: Two random words (TitleCase)
- Examples: `QuickTask`, `TempWork`, `BrightIdea`
- Word pools: colors, animals, adjectives, nouns

### Session Persistence

**Storage:** Config file (`~/.config/app/sessions.json`)

**Format:**
```json
{
  "sessions": {
    "WorkProject": "uuid-abc-123",
    "Research": "uuid-def-456"
  }
}
```

**Auto-save when:**
- Session starts with explicit name
- Environment variable set
- Session created with flag

### Session Resumption

**By name:**
```bash
cli --resume WorkProject
```
Looks up ID from config, restores session

**By ID:**
```bash
cli --resume uuid-abc-123
```
Directly restores by ID

**From file:**
```bash
cli --resume-from session.save
```
Extracts ID from filename or content

### Terminal Integration

**Auto-update tab/pane names** when:
- Starting new session
- Resuming existing session
- Switching between sessions

**Detection:**
- Checks for multiplexer env vars (`$TMUX`, `$ZELLIJ`, etc)
- Uses appropriate command for each multiplexer
- Graceful fallback if not in multiplexer

### Environment Variables

**Exports for downstream tools:**
- `SESSION_ID` - Current session UUID
- `SESSION_NAME` - Human-readable name

**Reads from environment:**
- `SESSION_NAME` - Default session name
- Cleared on resume to avoid conflicts

---

## Session Lifecycle

```bash
# Create
cli --session-name NewWork
# → Generates ID, saves mapping, exports vars

# Work
cli --continue
# → Resumes last session

# Resume later
cli --resume NewWork
# → Restores by name

# List all
cli --list-sessions
# → Shows all saved sessions

# Clean up
cli --delete-session OldWork
# → Removes from storage
```

---

## Error Handling

**Session not found:**
```bash
cli --resume NonExistent
# Shows: ❌ Session 'NonExistent' not found
#        Available: WorkProject, Research, TempWork
```

**No sessions saved:**
```bash
cli --list-sessions
# Shows: No saved sessions
#        Start with --session-name to create one
```

**Invalid file:**
```bash
cli --resume-from missing.save
# Shows: ❌ File not found: missing.save
```

**Duplicate name:**
```bash
cli --session-name WorkProject
# Shows: ⚠️  Session 'WorkProject' exists (uuid-123)
#        Continue existing? [y/N]
```

---

## Tab Completion

Sessions available in shell completion:
```bash
cli --resume <TAB>
# Shows: WorkProject  Research  TempWork
```

Config path completion:
```bash
cli --resume-from <TAB>
# Shows: *.save files in current dir
```
