# agent-session

Manage saved AI agent sessions across providers (Claude, Pi, etc.).

Dual-surface: **CLI commands always output JSON** for agents and scripts. **TUI** (`cly as` or `cly as tui`) for interactive use.

## Quick Start

```bash
cly as                          # Open interactive TUI (current dir)
cly as -a                       # TUI with all sessions
cly as ls                       # List sessions as JSON
cly as upsert <id>              # Create or update a session
cly as rm <name>                # Delete a session
cly as resume <name|id>         # Resume a session
cly as edit <name>              # Edit name/description
```

## Scoping

All commands scope to the **current directory** by default. Use flags to change:

| Flag | Effect |
|------|--------|
| `-a` / `--all` | All sessions, any directory |
| `--directory /path` | Scope to a specific directory |
| `-p <provider>` | Filter by provider (claude, pi, all) |

These are persistent — they work on every subcommand.

## Commands

### `ls` — List sessions

```bash
cly as ls                       # Current dir, JSON array
cly as ls -a                    # All sessions
cly as ls --filter "deploy"     # Name substring match (case-insensitive)
cly as ls -a -p pi              # All Pi sessions
```

Output is always JSON:
```json
[
  {
    "name": "my-session",
    "provider": "claude",
    "saved_at": "2026-03-23 14:30",
    "id": "abc-123",
    "path": "/Users/yuri/project",
    "description": "refactoring work",
    "meta": {
      "env": "prod"
    }
  }
]
```

### `upsert` — Create or update a session

**`<id>` is the only required argument.** Name and description are optional.

```bash
# Minimal — just an ID
cly as upsert abc-123

# With name and description (positional)
cly as upsert abc-123 my-session "working on feature X"

# With flags
cly as upsert abc-123 --name my-session --description "feature X"

# Arbitrary key-value metadata
cly as upsert abc-123 --set env=prod --set team=infra
cly as upsert abc-123 --meta '{"env":"prod","team":"infra"}'
```

Behavior:
- **Creates** if no session with that ID exists
- **Updates** if found — merges metadata, overwrites other fields
- **`saved_at`** updates on every upsert
- **Always returns the full entry** as JSON

```json
{
  "id": "abc-123",
  "name": "my-session",
  "provider": "claude",
  "path": "/Users/yuri/project",
  "description": "feature X",
  "saved_at": "2026-03-23T14:30:00Z",
  "meta": {
    "env": "prod",
    "team": "infra"
  }
}
```

**Alias:** `save` works as an alias for `upsert`.

### `rm` — Delete sessions

```bash
# By exact name
cly as rm my-session

# By name filter (deletes all matching)
cly as rm --filter "old-deploy"

# Preview without deleting
cly as rm --filter "old" --dry-run
```

Output:
```json
{"deleted": [{"name": "old-deploy-1", "provider": "claude", "id": "..."}]}
```

`--dry-run` returns `would_delete` instead of `deleted`.

Cannot use both `<name>` and `--filter` at the same time.

### `edit` — Edit name/description

```bash
cly as edit my-session          # Opens interactive form
cly as edit                     # Opens picker, then form
```

Returns the full updated entry as JSON.

### `resume` — Resume a session

```bash
cly as resume my-session        # By name
cly as resume abc-123           # By ID
```

Execs into the provider's CLI (e.g., `claude -r <id>`). No JSON output — replaces the current process.

### `tui` — Interactive manager

```bash
cly as tui                      # Same as `cly as`
cly as tui -a                   # All sessions
```

Keybindings:

| Key | Action |
|-----|--------|
| `enter` | Resume selected session |
| `space` | Toggle select for bulk operations |
| `a` | Select all / deselect all |
| `d` | Delete selected (with confirmation) |
| `e` | Edit selected session |
| `s` | Cycle sort order (date ↓↑, name ↓↑) |
| `y` | Toggle yolo mode (if provider supports it) |
| `/` | Filter by name |
| `q` | Quit |

## Data Model

Sessions are stored in `~/.config/cly/sessions.json`.

```go
type Entry struct {
    ID          string            `json:"id"`
    Name        string            `json:"name,omitempty"`
    Provider    string            `json:"provider,omitempty"`
    Path        string            `json:"path"`
    Description string            `json:"description,omitempty"`
    SavedAt     time.Time         `json:"saved_at,omitempty"`
    Meta        map[string]string `json:"meta,omitempty"`
}
```

- **ID** — session identifier (required for upsert)
- **Name** — human-friendly label (optional)
- **Provider** — `claude`, `pi`, etc.
- **Path** — directory where the session was created (used for scoping)
- **Meta** — arbitrary key-value pairs via `--set` or `--meta`

## Providers

Providers are configurable in `~/.config/cly/config.yaml`:

```yaml
modules:
  agent_session:
    providers:
      claude:
        command: claude
        resume_args: ["-r", "{id}"]
        yolo_args: ["--dangerously-skip-permissions"]
      pi:
        command: pi
        resume_args: ["--session", "{id}"]
```

Default providers: `claude` and `pi`.

## File Layout

```
modules/agent-session/
├── cmd.go              # Command registration, persistent flags, helpers
├── ls.go               # ls subcommand (JSON list)
├── rm.go               # rm subcommand (delete with --filter, --dry-run)
├── upsert.go           # upsert subcommand (create/update with --set/--meta)
├── edit.go             # edit subcommand (interactive form, JSON output)
├── resume.go           # resume subcommand (exec into provider)
├── tui.go              # TUI (Bubbletea: list, delete, edit, resume)
├── picker.go           # Legacy picker (used by edit fallback)
├── sessions.go         # Data model, load/save, filtering
├── providers.go        # Provider config and exec logic
└── *_test.go           # Tests (68 passing)
```
