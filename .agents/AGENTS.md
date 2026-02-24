# .ai/ Folder Specification

**Purpose:** AI-agnostic context layer. Single source synced to all AI tools.

---

## Structure

```
.ai/
├── commands/     Slash commands for AI tools
├── skills/       Domain expertise (testing, TUI, Go patterns)
├── agents/       Advisory specialists (go-specialist)
└── drafts/       Rough feature specs (opinionated, HOW-focused)
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

**drafts/** - Rough specs
- Pass to OpenSpec instead of simple strings
- Opinionated (includes HOW, not just WHAT)
- Less formal than OpenSpec proposals
