# Change: Add cly diff — bead-capture-while-reviewing-working-tree

## Why

Reviewing your own working tree before commit is a capture moment:
"I see something sketchy here, bead it, keep reading."

Existing tools (gh pr view, delta, lazygit) show diffs. None let you
attach bead issues to files during review, track what's read, or pick
up where you left off when files change.

`cly diff` fills that niche: a **local web UI** that shows your working
tree diff, tracks read state per file, invalidates on edit, and lets
you fire beads via `bd` with one keystroke.

## Scope

**In scope (v1, minimal):**
- `cly diff` command spawns local HTTP server, opens browser
- Shows `git diff HEAD` files (unstaged + staged + untracked)
- Per-file "mark as read" toggle
- "New bead" form that shells to `bd create`
- Go:embed React+Vite bundle — single binary

**Out of scope (v1):**
- Review state persistence (read/unread) — comes in v2
- fsnotify-based stale detection — comes in v2
- SSE push of file changes — comes in v2
- Line-range beading (select lines → N) — comes in v2
- Existing-beads badge on file — comes in v3
- AI hunk summary / auto-labels — blocks on helpy fix
- Commit walking / PR review / range diff — not this change
- Binary file preview (skip from list)

## Philosophy

- **UI shell over bd CLI**: no new bead storage. `bd create --json`, `bd label list-all --json`. Degrade gracefully if no `.beads/` DB.
- **Local-only**: bind 127.0.0.1. No auth token — localhost trust model.
- **Single binary**: React `dist/` embedded via go:embed. `task build` runs `npm run build` first.
- **Don't touch mockup**: `.agents/tmp/cly-diff-mockup.html` stays as design reference. Rewrite in React+TS from scratch.

## Commands

```bash
cly diff                       # serve + open browser
cly diff --port 54771          # pick port explicitly
cly diff --no-open             # serve only, print URL
```

Default port: pick uncommon free port, print URL. Ctrl+C to exit.

## UX flow (v1)

```
$ cly diff
  cly diff serving at http://127.0.0.1:54771
  opening browser…

  ┌─ browser ────────────────────────────────┐
  │ files with changes:                      │
  │   ● modules/helpy/streaming.go  +4 -0    │
  │   ● pkg/config/config.go        +1 -1    │
  │   ● cmd/root.go                 +3 -2    │
  │                                          │
  │ click file → render diff                 │
  │ press 'n' → new bead modal               │
  │   (title, desc, type, priority, labels)  │
  │   context auto-filled = current file     │
  │ submit → POST /api/bead                  │
  │   → shell `bd create --context <path>`   │
  │   → toast "bead bd-42 created"           │
  │                                          │
  │ [mark as read] button per file           │
  │ empty diff → "nothing to review"         │
  └──────────────────────────────────────────┘

  ^C → server shuts, cly exits
```

## bd integration

Shell out, no library coupling:

```
GET  labels        → bd label list-all --json
POST bead          → bd create --title X --type X --priority X \
                       --labels A,B --context <file> --json
                   → returns bead ID
```

Missing `.beads/` DB: frontend shows warning banner, disables bead
button, rest of review UI still works.

## Degradation

```
Condition                    Behavior
─────────────────────────────────────────────────────
git not installed            cly diff exits with error
not in git repo              cly diff exits with error
empty diff                   UI shows "nothing changed"
                             + global bead option
bd not installed             UI works, bead btn disabled
no .beads/ DB                UI works, bead btn disabled
                             + banner "run bd init"
binary files in diff         skipped from review list
huge diff (10k+ lines)       virtual scroll (v2+)
port taken                   fail, suggest --port flag
```

## What changes

- New module: `modules/diff/`
- New spec capability: `diff-reviewer`
- New dev dep: npm/Vite pipeline under `modules/diff/web/`
- New Taskfile target: `task build:web`
- Register command in `cmd/root.go`

## Non-goals

- Not a diff viewer replacement (delta, difftastic stay in use)
- Not a PR review tool (no remote fetching)
- Not a git client (no commit, no stage, no push)
- Not tied to specific bd DB — plug in via CLI
