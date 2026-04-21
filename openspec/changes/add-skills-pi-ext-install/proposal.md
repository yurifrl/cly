# add-skills-pi-ext-install

## Problem Statement
cly currently has no way to distribute opinionated AI agent content to users. Two related gaps:

1. **Skills** — AI agent skills (SKILL.md docs that teach an agent how to behave) live only in personal setups like `~/.agents/skills/` or Dotfiles. cly, which is standalone and open source, has no mechanism to ship its own curated skills to users.
2. **Pi extensions** — pi (a TUI coding agent) supports TypeScript extensions, but cly offers no way to install extensions that integrate cly with pi. In particular, there is no `/save` slash command in pi that drives the existing `cly gs save` command.

The result is friction: users who want cly to feel native in pi (or want cly-authored guidance available to their agents) have to hand-wire everything.

## Proposed Solution
Add two install commands to cly, each backed by a `//go:embed` folder of in-repo authored assets:

- `cly skills install` → copies embedded skills to `~/.agents/skills/`
- `cly pi extensions install` → copies embedded pi extensions to `~/.pi/agent/extensions/`

Ship two initial assets:

- Skill **`agents-session`** — guidance for agents running a session; teaches the agent (best-effort) to call `cly gs save` with `id`, `name`, `description`.
- Pi extension **`save`** — TypeScript extension that registers a `/save` slash command in the pi TUI. Determines `id`, `name`, `description` in code and invokes `cly gs save`. Argument parsing allows positional name override and `description="..."` kv override.

Both installers overwrite existing content by default and print every action (written / overwritten / skipped). Support `--target <dir>` and `--dry-run`.

## Success Criteria
- `cly skills install` copies `agents-session` to `~/.agents/skills/agents-session/SKILL.md` (overwrite + verbose output).
- `cly pi extensions install` copies `save` to `~/.pi/agent/extensions/save/` (overwrite + verbose output).
- `--dry-run` and `--target` work on both.
- In pi, `/save`, `/save some name`, and `/save name here description="foo"` all call `cly gs save` with the expected arguments.
- Dotfiles can drive installation with `@once` entries in `dotfiles.conf`.
- Integration tests pass against tmp target dirs.

## Scope

**In scope**
- New module `modules/skills/` with `install` subcommand + `embedded/agents-session/`
- New module `modules/pi/` with `extensions install` subcommand + `embedded/save/`
- `agents-session` SKILL.md authored in-repo
- `save` pi extension authored in-repo (TypeScript)
- Registration in `cmd/root.go`
- Integration tests for both installers

**Out of scope**
- `cly gs save` itself (assumed to exist or be handled separately)
- Skill/extension management beyond install: no add/update/remove/list/sync
- Non-embedded sources (git URLs, tarballs) — can come later
- MCP exposure of skills
- Symlinking to `~/.claude/skills` or other tool dirs (users can do this in Dotfiles)

## Dependencies
- `cly gs save` command must exist for the `/save` extension and skill guidance to be useful at runtime. It is not blocked by this change to ship, but it is blocked to be *useful*.
- Pi extension API — confirmed from the installed pi distribution at implementation time.

## Risks & Considerations
- **Overwrite surprises** — default overwrite could clobber user edits to installed skills. Mitigation: verbose output shows exactly what changed; `--dry-run` available; users can edit downstream copies and re-apply out-of-band.
- **Pi extension API drift** — pi's extension API is not versioned here. Mitigation: keep the extension thin, shell out to `cly` binary, confirm API at build time.
- **Path assumptions** — `~/.agents/skills/` and `~/.pi/agent/extensions/` are hard-coded defaults. Mitigation: `--target` flag.
- **Skill best-effort behavior** — the `agents-session` skill relies on the agent to extract values correctly. Acceptable per product intent (skill = best-effort, extension = deterministic).
