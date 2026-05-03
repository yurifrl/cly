# Capability: diff-reviewer

## Purpose

Local web UI to review `git diff HEAD` of the working tree and attach
`bd` beads to changed files during review.

## Behavior

### Command

`cly diff` starts a local HTTP server bound to 127.0.0.1 on a random free
port, prints the URL to stdout, and opens the user's default browser.
Ctrl+C stops the server and exits.

Flags:
- `--port N` — bind specific port (fail if taken)
- `--no-open` — start server, do not open browser

### Prerequisites

- `git` must be on PATH and current dir must be inside a git working tree.
- If missing, `cly diff` exits non-zero with a clear error.

### Diff list

On `GET /api/diff`:
- Returns all files changed vs HEAD (unstaged + staged).
- Includes untracked files (shown with status "untracked").
- Binary files (detected via "Binary files … differ" marker) included
  in response with `binary: true`; the UI hides them from the review list.
- Additions/deletions counts from `git diff --numstat`.
- Empty diff returns `{ files: [] }` — frontend shows "nothing to review".

### File diff

On `GET /api/diff/file?path=X`:
- Returns parsed hunks for that file (unified diff form).
- 404 if path is not in the current diff.
- 415 if binary.

### Labels

On `GET /api/labels`:
- Returns known labels from `bd label list-all --json`.
- If `bd` is missing or no `.beads/` DB exists, returns `{ labels: [] }`
  with HTTP 200 (graceful degradation).
- Frontend still allows typing new labels (free input).

### Bead creation

On `POST /api/bead` with JSON body `{ title, description, type, priority,
context, labels[] }`:
- Shells to `bd create ... --json` with arguments mapped from body.
- Returns `201 { id: "bd-NN" }` on success.
- Returns `409` if `bd` is missing or `.beads/` does not exist,
  with body `{ error: "no bd db" | "bd not installed" }`.
- Returns `400` on validation errors (empty title, invalid type/priority).
- Returns `500` on unexpected bd errors, with stderr in body.

### Frontend UX

- File list sidebar shows every non-binary file from the diff list.
- Click a file → render diff hunks on main panel.
- Keyboard: `n` opens bead modal pre-filled with context = current file.
- Bead modal fields: title (required), description, type (bug/feature/task/
  chore/decision), priority (P0–P4), context (defaults to file; can be
  changed to "global" or another file in diff), labels (chip input with
  suggest dropdown, arrow up/down to navigate).
- Submit → POST /api/bead → success toast with bead ID → modal closes.
- `Esc` closes modal. `Cmd+Enter` / `Ctrl+Enter` submits.
- Banner at top if `/api/health` reports `beadsDb: false`:
  "bd database not found. Run `bd init` to enable bead creation."
  Bead button disabled in that state.
- Empty diff shows "nothing to review" card with a "create global bead"
  button (bypasses file context).

### Single binary

- React frontend bundle is embedded via `go:embed` at build time.
- No runtime filesystem dependency for UI assets.
- `task build` runs `npm run build` before `go build`.

### Security posture

- Server binds `127.0.0.1` only — never `0.0.0.0`.
- No authentication — localhost-only trust model for v1.
- No CSRF token — localhost same-origin assumed.

## Non-goals

- Not a replacement for `delta`, `difftastic`, or `lazygit`.
- Does not walk git history, review PRs, or fetch from remotes.
- Does not persist review state across sessions in v1.
- Does not auto-reload on file changes in v1.
- Does not render binary file previews.
- Does not provide AI hunk summaries or auto-labels.

## Future scope (explicitly deferred)

- Per-branch review state in `.cly/review-state.json`
- Stale detection via `git hash-object` + fsnotify
- SSE push of file-changed events to browser
- Line-range beads (`context: path:L42-47`)
- Existing-beads badge per file
