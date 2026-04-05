---
created: 2026-04-05T00:00:00Z
project: cly
description: Rewrote charm-stack skill from scratch for Charm v2 (Bubbletea/Bubbles/Lipgloss/Huh)
context: .agents/skills/charm-stack/SKILL.md
tags: [charm, bubbletea, lipgloss, bubbles, huh, v2, skill]
session_name: charm-stack-v2-skill-rewrite
purpose: Audit and rewrite the charm-stack skill to cover Charm v2 APIs released Feb 2026
session_id: 2df1782c-6f24-4bc6-8be4-a3d56292c3c1
provider: pi
resume_with: cly agent-session resume --provider pi 2026-04-04-2100-oi-spell-checker
context_name: 2026-04-04-2100-oi-spell-checker
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-04-04-2100-oi-spell-checker.md
---

## Session
- **Name:** charm-stack-v2-skill-rewrite
- **Purpose:** Rewrite `.agents/skills/charm-stack/SKILL.md` for Charm v2
- **Resume:** `cly agent-session resume --provider pi 2026-04-04-2100-oi-spell-checker`

## Context
The charm-stack skill taught entirely v1 patterns (old import paths, `View() string`, `tea.KeyMsg`, `lipgloss.AdaptiveColor`, positional Bubbles constructors). Charm v2 was released Feb 24, 2026 with major breaking changes across all four libraries.

## Problem
Skill was outdated — would produce code that doesn't compile against v2.

## Decisions
- Rewrote from scratch rather than patching (too many changes)
- Sourced v2 APIs from: release notes, official README docs via MCP, charm.land/blog/v2
- Included migration reference table at the bottom for quick v1->v2 lookup
- Kept CLY module integration section with v2 patterns
- Covered all v2-only features: declarative View, clipboard, env vars, keyboard enhancements, compositing, viewport gutters/highlights, progress color API, hyperlinks, underline styles

## Current State
Done. Skill rewritten at `.agents/skills/charm-stack/SKILL.md`. Already committed as `782ae43`.

Note: the project itself still uses v1 deps (bubbletea v1.3.10, bubbles v0.21.0, lipgloss v1.1.1). Migration of project code is a separate task.

## Next Steps
- Migrate project dependencies to Charm v2 (`go get charm.land/bubbletea/v2` etc.)
- Update all existing modules to use v2 APIs (View() tea.View, KeyPressMsg, etc.)
---
