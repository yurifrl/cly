# AI CLI - Open Questions & TODO

## Questions to Resolve

### Root CLAUDE.md Placement
Where does `CLAUDE.md` in `.ai/` root sync to?
- [ ] Option A: `.claude/CLAUDE.md` (standard)
- [ ] Option B: `/CLAUDE.md` in project root (non-standard)
- [ ] Option C: Both?

### AGENT.md Pluralization
Verify target naming:
- [ ] Claude: `CLAUDE.md` (singular) ✓
- [ ] Crush: `AGENTS.md` (plural?) - **NEEDS VERIFICATION**
- [ ] OpenCode: `AGENT.md` (singular) ✓
- [ ] Cursor: ?
- [ ] Codex: ?

### `ides/` Directory Design
Current "copy as-is" approach is messy.

**Proposed alternatives**:
1. [ ] Remove `ides/` entirely - use converter layer for everything
2. [ ] Make `ides/` explicit override - still goes through converter
3. [ ] Rename to `overrides/` - clearer intent
4. [ ] Keep as-is but document as "escape hatch"

## Implementation TODO

### Phase 1: Refactor
- [ ] Extract current logic into converter pattern
- [ ] Create base IDEConverter class
- [ ] Implement ClaudeConverter
- [ ] Implement OpenCodeConverter
- [ ] Implement CrushConverter
- [ ] Add tests for each converter

### Phase 2: Config
- [ ] Create `~/.ai.json` schema
- [ ] Implement config loader
- [ ] Add profile support
- [ ] Add per-IDE settings

### Phase 3: History
- [ ] Git history backend
- [ ] Auto-commit on sync
- [ ] Rollback command
- [ ] Diff command

### Phase 4: Daemon
- [ ] File watcher (fsevents on macOS)
- [ ] Conflict detection
- [ ] Conflict resolution strategies
- [ ] Daemon CLI commands (start, stop, status)
