# AI CLI - Features

## Current Features

### Core Sync
- **Bidirectional sync**: `.ai` → IDE-specific directories (`.claude`, `.opencode`, `.crush`)
- **Two-phase sync**: Shared configs first, then IDE-specific overrides from `ides/<ide>/`
- **Scope**: Global (`~/.config/ai` or `~/.ai`) and local (`.ai`) configurations

### File Types
- **commands/**: CLI command definitions
- **agents/**: AI agent configurations
- **skills/**: Skill definitions (nested directories)
- **AGENT.md**: Project-level agent instructions
- **ides/**: IDE-specific overrides (copied as-is)

### JSONC Support
- Auto-interpolation with `${VAR}` environment variables
- Converts `.jsonc` → `.json` unless `@no-interpolation` comment present
- Evaluates env vars at sync time

### Backup & History
- **Backup to `/tmp`**: All overwritten/symlinked files saved before replacement
- **Session logging**: Timestamped operations in `/tmp/ai_sync_<timestamp>/log.txt`
- **Operation types tracked**:
  - `copied_files`: Regular file copies
  - `interpolated_jsonc`: JSONC → JSON conversions
  - `translated_skills`: Skill content transformations
  - `removed_files`, `removed_symlinks`: Pre-sync backups
  - `pruned_files`, `pruned_directories`: Pruned items

### Translation Layer
| Source | Target (Claude) | Target (OpenCode) |
|--------|-----------------|-------------------|
| `AGENT.md` | `CLAUDE.md` | `AGENT.md` |
| `commands/` | `commands/` | `command/` |
| `agents/` | `agents/` | `agent/` |
| `skills/` | `skills/` | `skills/` |
| `claude.json` | `settings.json` | — |
| `opencode.json` | — | `opencode.json` |

### CLI Interface
- `ai`: Sync global + local
- `ai -g`: Global only
- `ai -i <ide>`: Specific IDE(s)
- `ai -p`: Prune stale files (with confirmation)
- `ai -n`: Dry-run mode

## Desired Features (Not Yet Implemented)

### Daemon Mode
- Run as background daemon for continuous sync
- Conflict detection and resolution
- File system watching (inotify/fsevents)

### History
- Git-based change history per project
- Rollback to previous states
- Diff view between syncs

### Config
- `~/.ai.json`: Central config file
  - Global config paths
  - Default behavior (parse JSONC, enabled/disabled)
  - Per-IDE settings
  - Sync profiles

### Enhanced IDE Support
- Cursor, Codex, future IDEs
- Per-IDE conversion layer (one file per IDE, easy to maintain/test)
- Decoupled interpretation layer

### Pattern Standardization
- Use Claude patterns for skills/agents/commands
- Crush: `AGENTS.md` (plural) instead of `CLAUDE.md`
- Verify correct pluralization

### Architecture Improvements
- Fix `ides/*` "hack" - current design copies everything as-is
- Better separation of concerns
- Plugin-based architecture for new IDEs
