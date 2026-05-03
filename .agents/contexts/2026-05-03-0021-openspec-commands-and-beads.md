---
created: 2026-05-03T03:21:56Z
project: cly
description: Built `cly beads` inline Bubbletea TUI for `bd create` with pill pickers, quick-select, persisted state, and details toggle
context: New module `modules/beads/` plus hardening of `modules/aliases/` to prevent shadowing external binaries via cobra aliases
tags: [beads, tui, bubbletea, bubbles, picker, aliases, module]
session_name: 2026-05-03-0021-openspec-commands-and-beads
purpose: Ship a compact, powerful TUI that replaces `bd create --title= --description= --type=` with an inline form that remembers your last type/priority and shells out to bd on submit
session_id: 019dea79-4722-77c8-bc7d-a4918ba3a9ff
provider: pi
resume_with: cly agent-session resume --provider pi 2026-05-03-0021-openspec-commands-and-beads
context_name: 2026-05-03-0021-openspec-commands-and-beads
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-05-03-0021-openspec-commands-and-beads.md
---

## Session

- Name: 2026-05-03-0021-openspec-commands-and-beads
- Purpose: Build a simple-but-powerful inline TUI around `bd create` covering daily-use fields (title, description, type, priority, labels) plus optional details (acceptance, skills, context, design, notes) and a `--dry-run` toggle. Not alt-screen. Autocomplete for type. Persist last selection.
- Change ID: n/a (no OpenSpec proposal; direct module work)
- Resume: `cly agent-session resume --provider pi 2026-05-03-0021-openspec-commands-and-beads`

## Context

The user tracks work in [beads](https://github.com/NSXBet/bbt) via the `bd` CLI. The most common command is `bd create --title=... --description=... --type=task`, and typing those flags by hand is slow. They wanted a compact inline TUI with autocomplete for `type` — not a fullscreen app — that they can open instantly, fill in, and submit. It had to live in the existing `cly` modular Cobra/Bubbletea v2 codebase and follow the patterns established by `modules/demo/credit-card-form/` and `modules/demo/autocomplete/`.

A secondary problem surfaced mid-session: the `cly` alias generator (`modules/aliases/aliases.go`) emitted shell aliases for every cobra alias unconditionally, which meant registering `cly beads` with alias `bd` would have clobbered the real `bd` binary in fish.

## Problem

1. Compose `bd create` flags via an inline form without blocking the terminal or stealing the whole screen.
2. Autocomplete the `type` field against a curated list (`bug | feature | task | epic | chore | decision`), with letter-based quick-select (type `t` → jumps to `task`) instead of forcing users to type prefixes or cycle with arrows alone.
3. Remember the last-used `type` and `priority` across sessions.
4. Prevent `cly`'s Fish alias generator from shadowing the real `bd` binary.

## Decisions

- **New module `modules/beads/`** — registered via `cmd/root.go`. Command tree: `cly beads` (opens form), alias `bd` at the cobra level only, plus `new`/`create`/`n` subcommand.
- **Picker abstraction** (`modules/beads/picker.go`): reusable horizontal pill selector used for both `type` and `priority`. Letter quick-select with 800 ms TTL buffer cleared by a `tea.Tick` + generation counter (stale ticks ignored). Tagged `quickSelectExpireMsg` so the type picker's expire never clears the priority picker's buffer.
- **Rejected `textinput.ShowSuggestions`** — confirmed by reading `~/go/pkg/mod/charm.land/bubbles/v2@v2.1.0/textinput/textinput.go` that suggestions only render when the current value is a strict prefix of a suggestion. Pre-filled `task` → zero ghost text, arrows did nothing, UX dead. Replaced with a custom pill picker.
- **State file:** `~/.config/cly/beads/state.json` — `{last_type, last_priority}`, best-effort JSON read/write, never blocks submission. Written only on successful non-dry-run `bd create`.
- **Submit key `ctrl+enter`** — requires Kitty keyboard protocol (iTerm2, WezTerm, Ghostty, kitty, foot, Alacritty enhanced). Documented in code comment. Bubbletea v2 requests basic key disambiguation by default so no extra program option needed.
- **`ctrl+d` details toggle** — tier-2 fields (acceptance, skills, context, design, notes) live under a collapsible section. Tab order adapts based on `detailsOpen`. If you collapse while focused on a hidden field, focus bounces back to `title`.
- **`ctrl+r` dry-run toggle** — global; header shows `DRY-RUN` badge. Dry-run does NOT persist state (you're previewing).
- **`aliases.AnnotationSkipAlias`** — new exported constant `cly.alias.skip` on `modules/aliases/aliases.go`. Commands set the annotation to suppress both primary and cobra alias emission to fish. Chosen over a PATH-shadow check for cobra aliases (user picked option 2 explicitly — "2 is google").
- **Dropped `fNoInherit` / `--no-inherit-labels`** — dead weight without a `--parent` field (tier 3, not implemented). Removed field, keybind, `boolView` helper, and flag wiring.
- **Filtered `bd types --json`** through `allowedTypes` so the picker never grows past the curated 6 core types. Custom types (via `bd config set types.custom`) pass through unfiltered.

## Current State

Done. Build + vet green. Not committed. The agent-session test failures visible in `go test ./modules/...` are pre-existing and unrelated.

- `modules/beads/cmd.go` — Cobra wiring, exec wrapper that prints created ID and forwards bd stderr (~55 lines).
- `modules/beads/beads.go` — model, Init/Update/View, submit-to-bd pipeline (~460 lines).
- `modules/beads/picker.go` — reusable pill picker with tag-scoped quick-select (~145 lines).
- `modules/beads/state.go` — JSON state file (~55 lines).
- `modules/aliases/aliases.go` — added `AnnotationSkipAlias` constant + opt-out check.
- `modules/aliases/aliases_test.go` — `TestSkipAnnotationDisablesAllAliases` added (6 tests total, all pass).
- `cmd/root.go` — registers `beads.Register`.

Verified via `/tmp/cly-beads completion fish`: neither `alias beads` nor `alias bd` appears in generated fish output.

## Next Steps

- **No tests yet** for `picker` (quick-select buffer semantics, firstPrefixMatch edge cases) or `state` (JSON round-trip, missing file). Worth adding if this code stays long-term.
- **Tier-2 fields** tested manually only; acceptance/design/notes textareas currently all 3-row fixed height — may need SetHeight tuning if long content is common.
- **Tier-3 fields** (`--parent`, `--deps`, `--due`, `--estimate`, `--assignee`, `--external-ref`, `--spec-id`) deliberately skipped. Add to the details section only if daily usage demands.
- **Alternative submit key** for terminals that don't support Kitty keyboard protocol (macOS Terminal.app users will silently see nothing on `ctrl+enter`). Could add a fallback like `alt+enter` or a visible submit button — user hasn't asked.
- **Old state file** `~/.config/cly/beads/last-type` (plain text, from earlier iteration) is orphaned if any user created one in dev — harmless, but could auto-migrate on first read.
- **Desktop notification / toast** after successful create — nice-to-have; `pkg/notify` exists.
