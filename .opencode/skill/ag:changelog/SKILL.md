---
name: ag:changelog
description: Generates and updates project CHANGELOG.md from git commits, conversation decisions, and architectural changes. Auto-invoke this skill when new features are added, features are removed, or architecture changes occur.
---

# Changelog Generator

Maintain a CHANGELOG.md that records what happened over time. Date-based entries, no versions, no semver, no boilerplate. Just content.

## Structure

Entries are grouped by date. Each date is a section. Time progresses downward (newest at top). Within a date, use Added/Changed/Removed categories only when needed — skip empty ones.

```markdown
# Changelog

## 2026-02-07

### Added
- ag:changelog skill — auto-maintains changelog from git + conversation context

### Changed
- Agent commands renamed from `ai:*` to `ag:*` prefix

## 2026-02-05

### Added
- cly bundle command — unified package management
```

No `[Unreleased]`, no version numbers, no links to keepachangelog, no summary sections. Just dates and entries.

## When to Auto-Invoke

Trigger automatically when:
- A new feature is added
- A feature is removed or deprecated
- Architecture changes
- A breaking change is introduced

## Workflow

### Step 1 — Determine Scope

```bash
git log --oneline --no-merges -20
git status --short
git diff --name-status
git diff --cached --name-status
```

- **New commits since last changelog date**: Add a new date entry (or update today's if it already exists)
- **No new commits, just uncommitted changes**: Update today's entry with current state

### Step 2 — Gather What Changed

1. **Git commits** since the last changelog date
2. **Uncommitted changes** — include in today's entry
3. **Conversation context** — decisions and rationale from the session
4. **Verify existence** — for every entry, confirm the file/feature actually exists now. Don't document ghosts.

### Step 3 — Reconcile

Read existing CHANGELOG.md before writing:

1. **Remove stale entries** — if something listed as "Added" no longer exists, delete it
2. **Remove reversed changes** — added then removed in same period = no entry
3. **Update descriptions** — if behavior changed since logged, fix the description
4. **Deduplicate** — multiple commits for one thing = one entry
5. **If today's date already exists** — update it in place, don't create a duplicate

The changelog must be truthful. Every entry must reflect current reality.

### Step 4 — Write

If no CHANGELOG.md exists, ask before creating.

Append/update entries. Present proposed changes to the user before applying.

## Rules

- **Date-based, not version-based** — no semver, no `[Unreleased]`, just YYYY-MM-DD
- **Truthful** — every entry must be verifiable against current codebase
- **Remove when gone** — deleted feature = delete its entry
- **Update today, don't duplicate** — if today already has an entry, update it
- **No fluff** — no boilerplate headers, no summary sections, no links, no preamble beyond `# Changelog`
- **No trivial changes** — skip typos, formatting, WIP commits
- **Deduplicate** — one feature = one entry regardless of commit count
- **Concise** — one line per change, rationale after dash if worth mentioning
