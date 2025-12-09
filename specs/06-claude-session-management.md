# Claude Session Management

**Goal**: Port session management features from claudext to `cly claude` command

**Source**: `~/DotFiles/home/.config/fish/functions/claudext.fish`

**Scope**: Session save/resume functionality with name mapping

---

## Features to Port

### 1. Session Save (`--save-session`)

**Behavior:**
- Get current Claude session ID from running session
- Save to `~/.config/cly/sessions.json` with a name
- Map: name → session ID
- Display confirmation with name and short ID

**How claudext does it:**
- Calls Claude CLI with special flags to get session ID as JSON
- Parses JSON response
- Saves to ~/.config/claudext.json

**Storage format:** JSON file mapping names to session IDs

### 2. Ultra Switch (`-us, --ultra-switch <name>`)

**Behavior:**
- Look up session ID by name in sessions.json
- Resume that session with Claude CLI
- Display which session is being resumed

**How claudext does it:**
- Reads sessions map from JSON config
- Finds session ID by name
- Executes Claude with --resume flag and session ID

### 3. Resume from File (`-rf, --resume-from <file.paid>`)

**Behavior:**
- Extract session ID from .paid filename
- Resume with that session ID
- .paid file naming: `{session-id}.paid` (ID before first dot)

**How claudext does it:**
- Parses filename, splits on dot, takes first part as session ID
- Executes Claude with --resume and extracted ID

### 4. Initialize Flag (`-i`)

**Behavior:**
- Used together with continue flags
- First: Initialize new session to get real session ID
- Then: Continue with that session
- If --name provided: save the session with name

**How claudext does it:**
- Detects both -i flag AND (-c or --continue or --resume)
- Runs Claude with initial prompt to get session ID
- Saves session if --name was provided
- Continues with the session

### 5. Named Session Auto-Save

**Behavior:**
- When --name flag is used (from default case in claudext)
- Automatically initialize session and save to sessions.json
- Set CLAUDE_SESSION_NAME environment variable

**How claudext does it:**
- Parses --name flag from args
- Generates random ColorAnimal if no value provided
- Initializes session, gets ID from JSON response
- Calls save function with name and ID
- Exports env var

---

## Random Name Generator

**Pattern**: ColorAnimal (two capitalized words concatenated)

**Word Lists:**
- Colors (14): red, blue, green, yellow, purple, orange, pink, cyan, magenta, lime, teal, navy, maroon, olive
- Animals (16): cat, dog, fox, wolf, bear, lion, tiger, shark, eagle, hawk, dove, owl, rabbit, deer, mouse, rat

**Examples:** RedWolf, BlueFox, GreenTiger, YellowEagle, PurpleDeer

**Total combinations:** 224

---

## CLY Command Structure

```
cly claude
├── save [name]               # Save current session
├── resume <name>             # Resume by name
├── resume-file <file.paid>   # Resume from .paid file
├── list                      # List saved sessions
├── forget <name>             # Remove saved session
└── name [NAME]               # Generate/set session name
```

**With flags:**
- `cly claude --name [NAME]` - Set session name, auto-save when session starts
- `cly claude -i -c` - Initialize then continue pattern

---

## Session Storage Location

**File:** `~/.config/cly/sessions.json`

**Contains:** Map of session names to full session UUIDs

---

## Integration with Claude CLI

The module wraps/calls Claude CLI:
- Reads session IDs via `claude --output-format json`
- Resumes sessions via `claude --resume {id}`
- All other Claude flags pass through

---

## Requirements

### Save Session
- Get session ID from Claude CLI JSON output
- Accept optional name (generate ColorAnimal if not provided)
- Write to sessions.json (create if doesn't exist)
- Atomic file writes (temp file + move)

### Resume by Name
- Read sessions.json
- Look up session ID by name
- Verify session exists
- Execute Claude with --resume flag

### Resume from File
- Parse .paid filename for session ID
- Extract ID (text before first dot)
- Execute Claude with --resume flag

### List Sessions
- Read sessions.json
- Display all saved sessions
- Show: name, short ID (first 8 chars), save timestamp

### Name Generation
- Use exact word lists from claudext
- Random selection from colors and animals
- Capitalize each word, concatenate

### Session Validation
- Check if session ID is valid UUID format
- Verify session exists in Claude's history before resuming

---

## Future Integration with claude-sessions

**claude-sessions provides:**
- Search Claude Code history (~/.claude/history.jsonl)
- Show session metadata (messages, duration, tokens)
- Resume with project directory navigation

**Potential merge:**
- Use sessions.json for named sessions (quick access)
- Use history.jsonl for search/metadata (full history)
- Combined command could show both named + recent sessions

**For later:** Could add `cly claude search`, `cly claude show` using history.jsonl

---

## Out of Scope

**Not porting:**
- Workspace navigation
- Folder creation
- Archive functionality
- Timestamp generation
- Session metadata/analytics
- Full history search
- Project directory navigation

**Just:** Save/resume sessions with name mapping, integrating the 4 claudext features

---

## Module Location

`modules/claude/` (utility module)
