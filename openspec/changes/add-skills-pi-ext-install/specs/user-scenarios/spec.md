# User Scenarios

## Scenario 1: First-time install via Dotfiles
**Actor:** Yuri (repo owner, running `dotfiles-apply`).

**Precondition:** cly is built and on PATH. `~/.agents/skills/` and `~/.pi/agent/extensions/` may or may not exist.

**Steps:**
1. `dotfiles.conf` contains:
   ```
   @once cly-skills  -- cly skills install
   @once cly-pi-ext  -- cly pi extensions install
   ```
2. User runs the Dotfiles apply command.
3. Both `@once` blocks fire.

**Expected:**
- `~/.agents/skills/agents-session/SKILL.md` exists with embedded content.
- `~/.pi/agent/extensions/save/` exists with the TS extension files.
- Console shows `wrote <path>` lines for each file installed.

## Scenario 2: Preview changes before reinstall
**Actor:** User with existing installed skills.

**Steps:**
1. User edits `~/.agents/skills/agents-session/SKILL.md` manually.
2. User runs `cly skills install --dry-run`.

**Expected:**
- No files on disk change.
- Console shows `would write <path>` lines for every embedded file, so the user sees exactly what a real install would overwrite.

## Scenario 3: Custom target
**Actor:** Power user installing skills into a worktree / alt profile.

**Steps:**
1. User runs `cly skills install --target /tmp/sandbox/skills`.

**Expected:**
- Files written under `/tmp/sandbox/skills/agents-session/SKILL.md`.
- Default `~/.agents/skills/` is untouched.

## Scenario 4: `/save` with no args from pi
**Actor:** Developer in the pi TUI.

**Steps:**
1. User types `/save` and hits enter.

**Expected:**
- Extension computes `id`, `name`, `description` in code.
- `cly gs save` is invoked with those three values.
- Output from `cly gs save` appears in the TUI.

## Scenario 5: `/save` with positional name
**Steps:**
1. User types `/save refactor notes`.

**Expected:**
- `name` is `refactor notes`.
- `id` and `description` are defaults.
- `cly gs save` runs with those args.

## Scenario 6: `/save` with description override
**Steps:**
1. User types `/save refactor notes description="what I learned about the auth bug"`.

**Expected:**
- `name` is `refactor notes`.
- `description` is `what I learned about the auth bug`.
- `id` is default.
- `cly gs save` runs with all three.

## Scenario 7: Agent uses `agents-session` skill
**Actor:** AI agent reading `~/.agents/skills/agents-session/SKILL.md` at session start.

**Steps:**
1. Agent follows the skill's guidance.
2. At an appropriate moment, agent invokes `cly gs save ...` with best-effort-derived id/name/description.

**Expected:**
- Skill file is present after `cly skills install`.
- Skill content is readable and actionable — agent can produce a valid `cly gs save` invocation.
- Failure modes (wrong args) fall back to `cly gs save`'s own error handling, not this change's concern.
