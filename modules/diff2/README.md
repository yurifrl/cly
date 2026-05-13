# cly diff2

Local web UI for reviewing the working-tree `git diff` and capturing
`bd` beads while you read.

## Run

```bash
cly diff2                   # pick free port, open browser
cly diff2 --port 54771      # fixed port
cly diff2 --no-open         # serve only, print URL
```

Ctrl+C stops the server.

## Flow

1. `cly diff2` starts a local HTTP server on 127.0.0.1 (random port).
2. Browser opens at `http://127.0.0.1:<port>`.
3. Left sidebar = files changed vs HEAD (staged + unstaged + untracked).
4. Click a file → hunks render on the right.
5. Press `n` or click "new bead" → modal with title, desc, type,
   priority, context (auto = current file), labels.
6. Submit → shells `bd create` → toast shows the new ID.

## Prereqs

- `git` on PATH, current dir is a git working tree
- Optional: `bd` CLI + a `.beads/` DB (`bd init`). Without bd, the diff
  viewer still works; bead button is disabled and a banner explains.

## Build

```bash
task build          # builds frontend (Vite) then Go binary
```

Or manually:

```bash
cd modules/diff2/web && npm install && npm run build
cd ../../.. && go build ./...
```

Frontend output (`modules/diff2/web/dist/`) is embedded via `go:embed`.
A placeholder `index.html` ships in the repo so `go build` works before
the first frontend build.

## Layout

```
modules/diff2/
├── cmd.go              cobra command + run()
├── server.go           net/http routes + listener
├── git.go              parse `git diff` output
├── bd.go               shell out to `bd` CLI
├── browser.go          cross-platform open URL
├── embed.go            //go:embed web/dist
├── *_test.go           unit + integration tests
└── web/
    ├── package.json
    ├── vite.config.ts
    ├── src/
    │   ├── main.tsx
    │   ├── App.tsx
    │   ├── api.ts           fetch wrappers
    │   ├── types.ts         mirror of Go JSON types
    │   └── components/
    │       ├── FileList.tsx
    │       ├── DiffView.tsx
    │       ├── BeadModal.tsx
    │       └── ChipsInput.tsx
    └── dist/                built bundle (embedded)
```

## API

Server-side routes:

```
GET  /api/health        { git, bd, beadsDb: bool }
GET  /api/diff          { files: [{path,status,additions,deletions,binary}] }
GET  /api/diff/file?path=X   { path, binary, hunks: [...] }
GET  /api/labels        { labels: [] }
POST /api/bead          → 201 { id } | 409 (no bd/db) | 400 (bad input)
```

All responses are JSON. Errors shape: `{ error: string }`.

## Security

- Server binds `127.0.0.1` only
- No auth token — localhost same-origin trust model
- Use `--port` to pin a value; otherwise a random free port is picked

## Scope (v1)

Done:
- List files changed vs HEAD + untracked
- Render hunks
- Bead modal via `bd create`
- Graceful degradation when bd/db missing
- go:embed frontend bundle

Deferred (v2+):
- Per-branch review state (read/unread) in `.cly/review-state.json`
- `fsnotify` auto-reload on working-tree changes
- Server-Sent Events for live updates
- Line-range beading (select lines, attach to range)
- "Existing beads on this file" badge

## Why "diff2"?

Name chosen to avoid collision with the in-flight `modules/beads/` TUI
form. Rename to `diff` in a later change once that module's role is
settled.
