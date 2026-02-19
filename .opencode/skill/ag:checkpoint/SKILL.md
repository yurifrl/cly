---
name: ag:checkpoint
description: Session checkpoint — saves context, outputs summary + changelog. Three-in-one wrap-up command.
---

# Checkpoint

Save session context, print summary, update changelog. One command to wrap up a session.

## What It Does

1. **Save** — writes session context to `.agents/contexts/` (continuation prompt for LLM)
2. **Summary** — prints conversation recap to stdout
3. **Changelog** — updates CHANGELOG.md with current changes

## Usage

```
/ag:checkpoint
/ag:checkpoint --compact    # Compact summary format
```

## Workflow

### Phase 1 — Save Context

Save current session state to `.agents/contexts/` as a briefing for an LLM to continue work.

1. **Analyze conversation and extract:**
   - Context: What we're working on, relevant files/paths
   - Problem: What we were solving (specific, not abstract)
   - Decisions: Choices made and reasoning
   - Current State: Where we left off, what's done/not done
   - Next Steps: What to pick up, open questions

2. **Generate filename**
   - Format: `{YYMMDDHHMMSS}-{slug}.md`
   - Timestamp: current time as YYMMDDHHMMSS (no separators)
   - Slug: summary in lowercase, spaces to hyphens, max 50 chars
   - Run: `fish -c "date '+%y%m%d%H%M%S'"` to get timestamp

3. **Check for existing contexts**
   - Run: `fish -c "ls .agents/contexts/*.md 2>/dev/null | tail -5"`
   - If similar work exists, ask: "Update existing or create new?"

4. **Write context** to `.agents/contexts/{timestamp}-{slug}.md`

5. **File format:**

```markdown
---
created: <ISO timestamp, e.g. 2025-01-30T14:23:15>
project: <project/repo name>
description: <one-line summary of what this context is about>
context: <free text - what this relates to: ticket, parent feature, epic, other contexts>
tags: []
---

# {Title}

## Context
{What we're working on, project, relevant files/paths with line numbers}

## Problem
{What we were solving - be specific, not abstract}

## Decisions
{Choices made and WHY - this is critical for continuity}
- Decision 1: chose X over Y because Z

## Current State
{Where we left off}
- Done: ...
- Not done: ...
- Blocked on: ... (if any)

## Next Steps
{What to pick up, in priority order}
1. First thing to do
2. Open questions to resolve
```

6. **Confirm saved:** print filename and path

#### Save Writing Guidelines

This context will be read by an LLM to continue work. Apply prompt engineering principles:
- **Be specific**: "auth module" → "src/auth/token.go:45-80, the refresh logic"
- **Include reasoning**: Don't just say what, say why decisions were made
- **Provide context**: File paths, line numbers, relevant code snippets if needed
- **Clear next steps**: Prioritized, actionable items
- **No fluff**: Every sentence should help the next session continue effectively

### Phase 2 — Print Summary

Output conversation summary to stdout.

#### What Gets Captured

**ALWAYS START WITH THE GOAL** — answer "what was this session about?" first.

1. **Goal/Context** (REQUIRED) — what the user wanted, why the session happened
2. **Work Done** (REQUIRED) — what actually happened, what was built/fixed/explored
3. **Outcome** (REQUIRED) — current state, what's ready, what's pending
4. **Key Decisions** (OPTIONAL) — trade-offs, alternatives considered
5. **Next Steps** (OPTIONAL) — clear follow-up actions

#### Standard Format
```
## Summary

Goal: <what the user wanted to accomplish>

Work Done: <what actually happened>

Outcome: <current state>

Key Decisions: <only if meaningful>

Next Steps: <only if clear follow-ups exist>
```

#### Compact Format (`--compact`)
Single paragraph: goal + work + outcome.

#### Summary Rules
- **Goal first** — always answer "what was this session about?"
- **Summary not changelog** — focus on what user wanted, not file-by-file changes
- **Why over what** — explain reasoning, not just choices
- **Skip minutiae** — don't list every file read or test written
- **Write for memory** — reader should understand session purpose
- **Concise** — be succinct

### Phase 3 — Update Changelog

Update CHANGELOG.md with current changes.

#### Step 1 — Determine Scope

```bash
git log --oneline --no-merges -20
git status --short
git diff --name-status
git diff --cached --name-status
```

- **New commits since last changelog date**: Add/update date entry
- **No new commits, just uncommitted changes**: Update today's entry

#### Step 2 — Gather What Changed

1. **Git commits** since last changelog date
2. **Uncommitted changes** — include in today's entry
3. **Conversation context** — decisions and rationale from session
4. **Verify existence** — confirm file/feature actually exists. Don't document ghosts.

#### Step 3 — Reconcile

Read existing CHANGELOG.md before writing:

1. **Remove stale entries** — "Added" but no longer exists = delete it
2. **Remove reversed changes** — added then removed = no entry
3. **Update descriptions** — if behavior changed, fix the description
4. **Deduplicate** — multiple commits for one thing = one entry
5. **If today's date exists** — update in place, don't duplicate

#### Step 4 — Write

If no CHANGELOG.md exists, ask before creating. Present proposed changes before applying.

#### Changelog Structure

```markdown
# Changelog

## YYYY-MM-DD

### Added
- feature — brief description

### Changed
- what changed — why

### Removed
- what was removed
```

#### Changelog Rules
- **Date-based** — no semver, no `[Unreleased]`, just YYYY-MM-DD
- **Truthful** — every entry verifiable against current codebase
- **Remove when gone** — deleted feature = delete its entry
- **Update today, don't duplicate**
- **No fluff** — no boilerplate, no preamble beyond `# Changelog`
- **No trivial changes** — skip typos, formatting, WIP commits
- **Deduplicate** — one feature = one entry
- **Concise** — one line per change

## Output Format

The final output printed to the user should be:

```
---
Context saved: .agents/contexts/{filename}.md
---

## Summary

{summary content}

---

## Changelog

{changelog entries that were added/updated}
```

## Rules

- Run all three phases in order: save → summary → changelog
- If changelog has nothing meaningful to add, skip it and note "No changelog updates"
- Save file is always written (that's the point of checkpoint)
- Summary is always printed
- Be concise across all phases — user can't stand verbose output
