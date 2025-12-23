---
triggers: [architecture, structure, why, decision, pattern]
---

# Architecture Decisions

## Module Registration Pattern

**What:** Each module exports `Register(parent *cobra.Command)`

**Why:** Zero coupling - modules don't import each other. Add command = 1 line.

**Code:** cmd/root.go:45-59, modules/uuid/cmd.go:8-17

## Zero Coupling

**What:** Modules ONLY import pkg/*, never modules/*

**Why:** Parallel development, modules are portable (copy folder = works)

**Test:** Can you move modules/uuid/ to new repo? Should work with just import path change.

## Config Precedence

**Order:** env > local.yaml > user.yaml > embedded

**Why:** Dev secrets (local), user prefs (user), safe defaults (embedded), runtime override (env)

## Store vs Config

**Store** (pkg/store) - Runtime state (installed packages). SQLite, mutable.
**Config** (pkg/config) - User preferences. YAML, loaded once.

**When:** State that changes = Store. Settings = Config.

## Integration Testing Default

**Why:** Integration tests with real deps beat mocked unit tests. <2 mocks rule.

**See:** .ai/skills/testing/SKILL.md

## Chromedp over Puppeteer

**Why:** Single binary (no Node.js), goroutines > promises, native Go control.

## SQLite for State

**Why:** ACID guarantees, queryable, no runtime deps (pure Go driver), single binary.
