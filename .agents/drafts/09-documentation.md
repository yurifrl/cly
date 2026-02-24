# CLY Documentation Plan - Phase 1

**Focus:** AI navigation efficiency, minimal noise, scannable for future-you

**Priorities:** Module context, Architecture decisions, Pattern library

---

## Important Distinctions

**Blueprints** = Implementation instructions for spec-driven tools (NOT documentation)
- Used by OpenSpec workflow as implementation guides
- Structured instructions for AI to build features
- Keep separate, do NOT modify

**Docs** = Documentation explaining what exists and why
- Context for understanding codebase
- Architecture decisions and patterns
- Module purpose and structure

---

## Overview

This plan adds AI-focused documentation to help Claude navigate the codebase efficiently and make correct decisions without filling context with unnecessary reading.

**Success Metrics:**
- Claude finds relevant context in <5 file reads
- Decisions documented (no repeated "why" questions)
- Patterns answer "how to X" quickly
- Future-you can skim any doc in 60 seconds

---

## Tasks (Resume from any point)

### ✓ Phase 0: Research & Planning (COMPLETE)
- [x] Analyze existing documentation
- [x] Research AI documentation patterns
- [x] Map architecture decisions
- [x] Get user input on priorities
- [x] Create this plan

---

### Phase 1: Quick Wins (2-3 hours)

#### Task 1.1: Create Documentation Index
**File:** `docs/INDEX.md`
**Time:** 20 min
**Purpose:** AI can scan all available docs quickly

```markdown
# CLY Documentation Index

## For AI Assistants
- `.ai/skills/` - Domain-specific capabilities
- `docs/decisions/` - Architecture decisions (ADRs)
- `docs/patterns/` - Common implementation patterns
- `modules/*/README.md` - Module context

## Implementation Tooling (NOT docs)
- `.ai/drafts/` - Implementation instructions for spec-driven tools
- `openspec/` - Formal specifications and change tracking

## For Humans (if they show up)
- `README.md` - Getting started
- `docs/` - Understanding the codebase

## Status Legend
✓ Current | ⚠ Needs update | 📦 Historical
```

**Resume Point:** If interrupted, start fresh (10 min task)

---

#### Task 1.2: Create ADR Directory
**Command:** `mkdir -p docs/decisions`
**Time:** 1 min

---

#### Task 1.3: Write ADR - Modular Registration
**File:** `docs/decisions/0001-modular-registration.md`
**Time:** 20 min
**Template:**
```markdown
---
name: modular-registration-pattern
triggers: ["Register()", "module coupling", "adding commands"]
domain: architecture
level: core
applies_when: ["Adding new CLI commands", "Understanding module structure"]
---

# ADR 0001: Modular Registration Pattern

**Status:** Accepted

**Decision:** Each module exports `Register(parent *cobra.Command)` function

**Context:**
Scale from 5 to 100+ commands without coupling. Traditional approach (centralized command definitions in root.go) creates tight coupling and merge conflicts.

**Why This Works:**
- Zero coupling between modules
- Add command = 1 line in cmd/root.go
- Modules testable in isolation
- Parallel development enabled

**Alternatives Rejected:**
- Global registry (implicit, hard to track)
- Plugin system (runtime overhead, complexity)
- Init() auto-discovery (hidden magic)

**Evidence:**
- `cmd/root.go:47-59` - Single registration point
- `modules/uuid/cmd.go:8-16` - Module self-registration
- `modules/demo/cmd.go:112-114` - Namespace pattern
```

**Resume Point:** Copy template, fill in details from research

---

#### Task 1.4: Write ADR - Zero Coupling
**File:** `docs/decisions/0002-zero-coupling.md`
**Time:** 20 min
**Key Points:** Why modules don't import modules, only pkg/

---

#### Task 1.5: Write ADR - SQLite for State
**File:** `docs/decisions/0003-sqlite-for-state.md`
**Time:** 20 min
**Key Points:** Why Store interface uses SQLite (single binary, ACID, pure Go)

---

#### Task 1.6: Write ADR - Charm Stack
**File:** `docs/decisions/0004-charm-stack.md`
**Time:** 20 min
**Key Points:** Why Bubbletea+Bubbles+Lipgloss (Elm architecture, working examples)

---

#### Task 1.7: Write ADR - TDD Integration First
**File:** `docs/decisions/0005-tdd-integration-first.md`
**Time:** 20 min
**Key Points:** Why integration tests default, <2 mocks rule

---

#### Task 1.8: Write ADR - Config Hierarchy
**File:** `docs/decisions/0006-config-hierarchy.md`
**Time:** 20 min
**Key Points:** local.yaml > config.yaml > embedded > env vars

---

#### Task 1.9: Enhance Skill Frontmatter
**Files:** All 9 `.ai/skills/*/SKILL.md`
**Time:** 30 min (batch edit)
**Add:** `triggers`, `applies_when`, `domain`, `level`, `antipatterns` fields

**Resume Point:** Process one file at a time, can stop/resume anywhere

---

#### Task 1.10: Archive Stale Docs
**Time:** 15 min
**Actions:**
- Move `TODO.md` → `docs/archive/TODO-2025-12.md`
- Add disclaimer to `docs/architecture.md` (mark phases as historical)
- Create `docs/archive/README.md` explaining archived content

---

### Phase 2: Module Context (4-6 hours)

Each module README = 30 min avg, can pause between any module

#### Task 2.1: MCP Module README
**File:** `modules/mcp/README.md`
**Focus:** Complex system (40 files), explain context detection, adapters, TUI

---

#### Task 2.2: Bundle Module README
**File:** `modules/bundle/README.md`
**Focus:** Package manager, Store usage, bundler interface

---

#### Task 2.3: Scraper Module README
**File:** `modules/scraper/README.md`
**Focus:** Browser automation, extractors, sequential vs parallel

---

#### Task 2.4: Dotfiles Module README
**File:** `modules/dotfiles/README.md`
**Focus:** Symlink management, config parsing, install commands

---

#### Task 2.5: Notify Module README
**File:** `modules/notify/README.md`
**Focus:** Hook system, multi-notifier, Zellij integration

---

#### Task 2.6: Update Module README
**File:** `modules/update/README.md`
**Focus:** Self-updater, version parsing, GitHub releases

---

#### Task 2.7: Config Module README
**File:** `modules/config/README.md`
**Focus:** Viper wrapper, subcommands, precedence

---

#### Task 2.8: Helpy Module README
**File:** `modules/helpy/README.md`
**Focus:** Markdown viewer, search, palette

---

#### Task 2.9: Claude Module README
**File:** `modules/claude/README.md`
**Focus:** Session management, Zellij integration

---

#### Task 2.10: AI Module README
**File:** `modules/ai/README.md`
**Focus:** Mods wrapper, conversation threading

---

#### Task 2.11: Backup Module README
**File:** `modules/backup/README.md`
**Status:** EXISTS, enhance with metadata frontmatter

---

#### Task 2.12: UUID Module README
**File:** `modules/uuid/README.md`
**Focus:** Interactive generator, v4/v7 support

---

#### Task 2.13: Demo Namespace README
**File:** `modules/demo/README.md`
**Focus:** Overview of 48 demos, reference to upstream examples, how they're organized

---

#### Task 2.14: pkg/config README
**File:** `pkg/config/README.md`
**Focus:** Viper integration, secret resolution, config struct

---

#### Task 2.15: pkg/store README
**File:** `pkg/store/README.md`
**Focus:** Store interface, SQLite implementation, namespace/key model

---

### Phase 3: Pattern Library (2-3 hours)

Each pattern = 20-30 min, self-contained

#### Task 3.1: Create Patterns Directory
**Command:** `mkdir -p docs/patterns`

---

#### Task 3.2: Bubbletea Module Pattern
**File:** `docs/patterns/bubbletea-module.md`
**Content:** When, Structure, Template, Critical rules, Examples

---

#### Task 3.3: Config Access Pattern
**File:** `docs/patterns/config-access.md`
**Content:** How modules read config, precedence, examples

---

#### Task 3.4: Store Usage Pattern
**File:** `docs/patterns/store-usage.md`
**Content:** When to use Store vs Config, interface usage

---

#### Task 3.5: Error Handling Pattern
**File:** `docs/patterns/error-handling.md`
**Content:** Partial success, fail gracefully, error wrapping

---

#### Task 3.6: Testing Module Pattern
**File:** `docs/patterns/testing-module.md`
**Content:** Table-driven, integration first, <2 mocks, file structure

---

### Phase 4: Finalization (1 hour)

#### Task 4.1: Update docs/architecture.md
**Time:** 20 min
**Action:** Add "Historical Context" section, mark completed phases

---

#### Task 4.2: Create docs/archive/README.md
**Time:** 10 min
**Content:** Explain what's archived and why

---

#### Task 4.3: Update Root README.md
**Time:** 15 min
**Action:** Add "Documentation" section linking to INDEX.md, decisions/, patterns/

---

#### Task 4.4: Final Validation
**Time:** 15 min
**Check:**
- All metadata fields consistent
- No broken links
- All files <500 words (modules), <400 words (ADRs), <300 words (patterns)
- Triggers/applies_when comprehensive

---

## Template Library

### Module README Template
```markdown
---
name: [module-name]
triggers: ["keyword1", "keyword2"]
applies_when: ["Use case"]
domain: [category]
---

# [Module Name]

**Intent:** One-line purpose

## Why This Exists
2-3 sentences explaining the problem solved

## Key Concepts
- **Concept 1:** Definition
- **Concept 2:** Definition

## Architecture
Brief structure overview (3-5 bullets)

## Files
List key files with 1-line purpose

## Common Tasks
How to use (3-5 examples)
```

### ADR Template
```markdown
---
name: [decision-name]
triggers: ["keyword1"]
domain: architecture
level: core
---

# ADR [NNNN]: [Title]

**Status:** Accepted | Rejected | Superseded

**Decision:** What we decided

**Context:** Why we needed to decide

**Why:** Reasoning (3-5 bullets)

**Alternatives Rejected:** What we didn't choose

**Evidence:** File references
```

### Pattern Template
```markdown
---
name: [pattern-name]
triggers: ["keyword1"]
applies_when: ["Use case"]
antipatterns: ["what-not-to-do"]
---

# [Pattern Name]

## When
Situation where this applies

## Structure
File/code organization

## Template
Minimal working example

## Critical Rules
Must-follow constraints

## Finding Examples
Where to look in codebase
```

---

## Progress Tracking

**Completed:** Phase 0
**Next:** Task 1.1 (Create INDEX.md)

**To Resume:** Read this file, find next unchecked task, execute

---

## Notes for Future Sessions

**If interrupted:**
1. Read this file to see progress
2. Find first unchecked task
3. Use template from "Template Library" section
4. Reference research in task description
5. Each task is independent - can skip/reorder within phase

**Quality over speed:**
- Keep docs SHORT (use word limits)
- Focus on AI findability (good triggers)
- Explain INTENT not mechanics
- Examples over prose
