# .ai/ Folder Specification

**Purpose:** AI-agnostic context layer. Single source synced to all AI tools.

---

## Structure

```
.ai/
├── commands/     Slash commands for AI tools
├── skills/       Domain expertise (testing, TUI, Go patterns)
├── agents/       Advisory specialists (go-specialist)
├── blueprints/   Rough feature specs (opinionated, HOW-focused)
└── docs/         Navigation and architecture
```

---

## What Goes Where

**commands/** - Executable commands
- release, openspec workflows
- Reusable command patterns

**skills/** - How to do things
- Testing (TDD, integration tests)
- Charm stack (Bubbletea, Bubbles)
- Module creation, Cobra patterns

**agents/** - Advisory only
- go-specialist (consults, doesn't implement)

**blueprints/** - Rough specs
- Pass to OpenSpec instead of simple strings
- Opinionated (includes HOW, not just WHAT)
- Less formal than OpenSpec proposals

**docs/** - Navigation
- Where things go (code structure)
- Why decisions made (architecture)
- Quick pattern references

---

## Sync to Tools

**.ai/** = source of truth (version controlled)

**Synced to:**
- `.claude/` (Claude Code)
- `.cursor/` (Cursor IDE)
- `.crush/` (future)
- `.opencode/` (future)

**Sync:** Run `scripts/ai-sync.sh` after updating .ai/

---

## When to Update

**Add to .ai/ when:**
- Teaching AI a new skill
- Documenting architecture decision
- Creating reusable command pattern
- Writing rough feature spec for OpenSpec

**Don't add:**
- Code tutorials (AI doesn't follow steps)
- API reference (code has that)
- What code clearly shows (focus on WHY)
