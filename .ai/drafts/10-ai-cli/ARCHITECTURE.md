# AI CLI - Architecture Design

## Principles
- **Heavily configurable**: Behavior driven by `~/.ai.json`
- **Decoupled**: IDE conversion layer separate from core logic
- **Single file per IDE**: Easy to maintain and test
- **Git as dependency**: Use git for history/backup instead of custom solution

## Directory Structure

```
.ai/                          # Source of truth
├── ai.json                   # Project config (ides to sync, options)
├── commands/                 # Shared commands
├── agents/                   # Shared agents
├── skills/                   # Shared skills
├── AGENT.md                  # Shared project instructions
├── CLAUDE.md                 # Root-level CLAUDE.md (synced where?)
├── opencode.json             # OpenCode-specific config
├── claude.json               # Claude-specific config
└── ides/                     # IDE-specific overrides (remove/rethink)
    ├── claude/
    ├── opencode/
    ├── crush/
    └── codex/

~/.ai.json                   # Global config
```

## Core Components

### 1. Config Loader
```
~/.ai.json
{
  "global_path": "~/.config/ai",
  "default_ides": ["claude", "opencode"],
  "parse_jsonc": true,
  "daemon": {
    "enabled": false,
    "watch_paths": [".ai", "~/.ai"],
    "conflict_resolution": "source|target|manual|newer"
  },
  "history": {
    "enabled": true,
    "backend": "git",
    "auto_commit": true
  },
  "profiles": {
    "full": {
      "ides": ["claude", "opencode", "crush", "codex"],
      "prune": true
    }
  }
}
```

### 2. IDE Conversion Layer (Per-IDE Files)

`converters/claude.py`:
```python
class ClaudeConverter:
    def map_subdir(name: str) -> str
    def map_agent_file(name: str) -> str
    def map_config_file(name: str) -> str | None
    def translate_skill(content: str) -> str
```

`converters/opencode.py`:
```python
class OpenCodeConverter:
    def map_subdir(name: str) -> str  # commands→command
    def translate_skill(content: str) -> str  # strip allowed-tools
```

### 3. Sync Engine
```
Source (.ai) → Converter → Target (.claude/.opencode/etc)
                      ↓
                 Git History
```

### 4. History Layer (Git-based)
```python
class GitHistory:
    def init_repo(project_path: Path) -> None
    def commit_sync(source: Path, target: Path, metadata: dict) -> None
    def rollback(target: Path, commit: str) -> None
    def diff(target: Path, from_commit: str, to_commit: str) -> None
```

### 5. Daemon Mode
```python
class Daemon:
    def watch(paths: list[Path]) -> None
    def detect_conflicts(source: Path, target: Path) -> list[Conflict]
    def resolve_conflict(conflict: Conflict, strategy: str) -> None
```

## Open Questions

### Root CLAUDE.md Placement
Where does `CLAUDE.md` in `.ai/` root sync to?
- Option A: `.claude/CLAUDE.md` (standard)
- Option B: `/CLAUDE.md` in project root (non-standard)
- Need clarification on expected behavior

### `ides/` Directory Design
Current "copy as-is" approach is messy:
- Pros: Simple, works for any IDE format
- Cons: No translation, bypasses converter layer

**Proposed alternatives**:
1. **Remove `ides/` entirely**: Use converter layer for everything
2. **Make `ides/` explicit override**: Still goes through converter
3. **Rename to `overrides/`**: Clearer intent

### AGENT.md Pluralization
Verify target naming:
- Claude: `CLAUDE.md` (singular)
- Crush: `AGENTS.md` (plural?) - needs confirmation
- OpenCode: `AGENT.md` (singular)

## Implementation Order
1. Refactor to converter pattern
2. Add git-based history
3. Implement daemon mode
4. Add `~/.ai.json` config system
