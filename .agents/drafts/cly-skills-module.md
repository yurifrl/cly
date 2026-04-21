# cly skills + pi extensions

Two install commands in cly shipping embedded assets; both the `agents-session` skill and the pi `/save` extension drive the existing `cly gs save` CLI.

> Note: This is a draft to organize ideas and scope before implementation.

## Goal
1. `cly skills install` — install embedded skills (starting with `agents-session`).
2. `cly pi extensions install` — install embedded pi extensions (starting with `/save`).
3. Both consumers call `cly gs save` under the hood.

## Architecture
```
cly repo
┌────────────────────────────────────────┐
│ modules/skills/embedded/               │
│   agents-session/SKILL.md              │  ──┐
│                                        │    │  both
│ modules/pi/embedded/                   │    ├─▶ call `cly gs save`
│   save/  (TypeScript pi extension)     │  ──┘
│                                        │
│ modules/skills/cmd.go                  │  → cly skills install
│ modules/pi/cmd.go                      │  → cly pi extensions install
└────────────────────────────────────────┘
```

Install targets:
- `~/.agents/skills/`
- `~/.pi/agent/extensions/`

Dotfiles:
```
@once cly-skills  -- cly skills install
@once cly-pi-ext  -- cly pi extensions install
```

## Components

**`cly skills install`**
- Copies embedded skills to `~/.agents/skills/`
- Overwrites by default, verbose, `--target`, `--dry-run`
- Initial set: `agents-session`

**`cly pi extensions install`**
- Copies embedded pi extensions to `~/.pi/agent/extensions/`
- Overwrites by default, verbose, `--target`, `--dry-run`
- Initial set: `save`

**Skill: `agents-session`**
- Instructs the agent on how to invoke `cly gs save`
- Agent does **best-effort** extraction of `id`, `name`, `description` from context and passes them as CLI args
- No guarantees — best-effort by definition

**pi extension: `/save`** (TypeScript)
- Slash command in pi TUI
- Determines `id`, `name`, `description` in TS code (deterministic, not best-effort)
- Invokes `cly gs save ...`
- Argument parsing:
  - `/save` — use prefilled values from TS
  - `/save some name here` — positional overrides `name`
  - `/save description="the description"` — kv overrides `description`
  - Combined: `/save some name here description="..."`

## Key Decisions
- Backend is `cly gs save`; both skill and extension are frontends to it
- Skill = AI best-effort; extension = deterministic code path
- Installers mirror each other: `modules/<x>/embedded/` + `cly <x> install`
- Overwrite by default, print actions
- `/save` syntax: positional name first, kv for other fields

## Open Questions
- None blocking — all prior questions resolved

## Implementation Notes
- Both modules follow `add-module` pattern
- `//go:embed embedded` + walk `embed.FS` to recreate tree at target
- pi extension (TypeScript): shells out to the `cly` binary; confirm pi slash-command API from actual pi install when coding
- Tests: integration for installers against tmp target dirs; assert files written + stdout actions
- `cly gs save` is assumed to already exist / be implemented separately — this draft does not cover it
