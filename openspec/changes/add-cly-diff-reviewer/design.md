# Design: cly diff

## Architecture

```
┌─ cly diff (Go process) ─────────────────────────────┐
│                                                     │
│  cobra cmd ──┐                                      │
│              ├─ HTTP server (chi, 127.0.0.1:PORT)   │
│              │    ├─ GET  /           → index.html  │
│              │    ├─ GET  /assets/*   → embedded    │
│              │    ├─ GET  /api/diff   → files+hunks │
│              │    ├─ GET  /api/labels → bd labels   │
│              │    └─ POST /api/bead   → bd create   │
│              │                                      │
│              ├─ git exec (os/exec)                  │
│              │    └─ `git diff HEAD --name-status`  │
│              │    └─ `git diff HEAD -- <file>`      │
│              │                                      │
│              ├─ bd exec (os/exec)                   │
│              │    └─ `bd label list-all --json`     │
│              │    └─ `bd create ... --json`         │
│              │                                      │
│              └─ browser launcher (xdg-open/open)    │
│                                                     │
│  go:embed dist/ (React build output)                │
│                                                     │
└─────────────────────────────────────────────────────┘

Browser ──── HTTP localhost ────→ cly diff process
         fetch JSON, render UI
```

## Module layout

```
modules/diff/
├─ cmd.go                   cobra cmd + flags
├─ server.go                chi router, handler wiring
├─ handlers.go              /api/* handlers
├─ git.go                   diff parsing, name-status, hunks
├─ bd.go                    shell out helpers
├─ browser.go               open URL cross-platform
├─ embed.go                 //go:embed dist/*
├─ embed_test.go            ensures dist/ present in build
└─ web/
   ├─ package.json
   ├─ tsconfig.json
   ├─ vite.config.ts
   ├─ index.html
   ├─ .gitignore            dist/, node_modules/
   ├─ src/
   │  ├─ main.tsx
   │  ├─ App.tsx
   │  ├─ api.ts             fetch wrappers, types
   │  ├─ types.ts           matches Go JSON shapes
   │  ├─ components/
   │  │  ├─ FileList.tsx
   │  │  ├─ DiffView.tsx
   │  │  ├─ BeadModal.tsx
   │  │  ├─ ChipsInput.tsx
   │  │  ├─ ContextPicker.tsx
   │  │  └─ EmptyState.tsx
   │  └─ hooks/
   │     └─ useKeyboard.ts
   └─ dist/                 built, embedded, gitignored
```

## Tech choices

| Aspect             | Choice                    | Why                              |
|--------------------|---------------------------|----------------------------------|
| Server router      | go-chi/chi                | lightweight, stdlib-ish          |
| HTTP bind          | 127.0.0.1 only            | no remote, no auth needed        |
| Port strategy      | random free + print       | no collision, no CSRF risk       |
| Frontend           | React 18 + Vite + TS      | user choice, fast, modern        |
| Styling            | CSS modules or plain CSS  | minimal; mockup CSS as reference |
| State mgmt         | useState + custom hooks   | small app, no Redux              |
| Transport          | fetch (req/resp)          | v1 — no SSE until v2             |
| Asset delivery     | go:embed dist/*           | single binary                    |
| Build pipeline     | `task build:web` first    | npm run build → dist/            |
| Git access         | os/exec to `git`          | zero CGO, no go-git dep          |
| Bd access          | os/exec to `bd` --json    | CLI stable contract              |

## API contract (v1)

```
GET /api/diff
  → 200 { files: [{path, status, additions, deletions, binary}] }
  → 200 { files: [] } when no diff
  → 500 on git error

GET /api/diff/file?path=<urlencoded>
  → 200 { path, hunks: [{header, lines:[{kind,old,new,text}]}] }
  → 404 if path not in diff
  → 415 if binary (frontend skips)

GET /api/labels
  → 200 { labels: ["backend","refactor",...] }
  → 200 { labels: [] } if bd unavailable (degrade)

POST /api/bead
  body: { title, description, type, priority, context, labels[] }
  → 201 { id: "bd-42" }
  → 409 { error: "no bd db" } → UI shows banner
  → 400 { error: "validation …" }

GET /api/health
  → 200 { git: true|false, bd: true|false, beadsDb: true|false }
```

## Port discovery

```go
l, err := net.Listen("tcp", "127.0.0.1:0")  // OS assigns free port
if err != nil { return err }
port := l.Addr().(*net.TCPAddr).Port
fmt.Printf("cly diff serving at http://127.0.0.1:%d\n", port)
```

`--port N` flag overrides; fails if taken (no fallback, predictable).

## Browser launch

```go
switch runtime.GOOS {
case "darwin":  exec.Command("open", url)
case "linux":   exec.Command("xdg-open", url)
case "windows": exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
}
```

`--no-open` flag skips (useful for SSH/remote/CI).

## Git parsing strategy

```
git diff HEAD --name-status -z          → file list + status (A/M/D/R)
git diff HEAD --numstat                 → +N -M per file
git diff HEAD -- <path>                 → unified diff for one file
git ls-files --others --exclude-standard → untracked files
```

Merge untracked + diff output into one file list.

Binary detection: parse `Binary files … differ` marker from diff output,
set `binary: true`, frontend filters them out (v1).

## bd integration details

```bash
# on startup
bd label list-all --json
  ok      → cache labels, feed suggestions
  error   → return empty, frontend banner

# on bead submit
bd create \
  --title "$TITLE" \
  --description "$DESC" \
  --type "$TYPE" \
  --priority "$PRIO" \
  --context "$FILE" \
  --labels "$L1,$L2" \
  --json
```

Parse JSON output, extract ID, return to frontend.

Error handling:
- `bd: command not found` → 409, frontend disables bead button
- `no beads database found` → 409, frontend banner + bead disabled
- any other non-zero exit → 500 + stderr in body

## Build pipeline

```yaml
# Taskfile.yml additions
tasks:
  build:web:
    dir: modules/diff/web
    cmds:
      - npm install
      - npm run build
    sources:
      - src/**/*
      - package.json
      - vite.config.ts
    generates:
      - dist/**/*

  build:
    deps: [build:web]
    cmds:
      - go build -o dist/cly .
```

Vite build outputs to `modules/diff/web/dist/`. Go embeds that path.

CI: runs `task build`, ensures `dist/` populated. No committed `dist/`.

## Deferred v2/v3 (for reference, NOT this change)

- Review state: `.cly/review-state.json` keyed by branch
- Stale detection: `git hash-object <file>` per file, compare to last-read hash
- fsnotify: watch tracked files, invalidate on change
- SSE: push invalidation events to browser (`text/event-stream`)
- Line-range beading: select lines in diff → N → context=`path:L42-47`
- Existing beads badge: `bd query "context:<path>"` per file, show count

## Alternatives considered

| Option                         | Rejected because                        |
|--------------------------------|-----------------------------------------|
| Bubbletea TUI instead of web   | keyboard nav + modal + fuzzy = pain     |
| htmx + vanilla JS              | user chose React explicitly             |
| Direct bd Go SDK (not CLI)     | tight coupling to bd internals          |
| WebSocket transport            | overkill, one-way push fits SSE         |
| Committed dist/ in repo        | stale risk, noisy diffs                 |
| Fixed port 8080                | collision, CSRF risk                    |
| Auth token in URL              | localhost only, browser same-origin OK  |

## Risks

- **npm in Go repo**: adds Node toolchain dep for contributors.
  Mitigation: `task build:web` shells npm. CI handles it. Docs say
  "install Node 20+".
- **Binary size**: embedded React bundle adds ~200KB to cly.
  Acceptable for feature weight.
- **bd CLI instability**: bd flags may change, breaks integration.
  Mitigation: pin bd version in docs, use `--json` output which
  is the stable contract.
- **Browser auto-open on SSH session**: opens wrong place.
  Mitigation: `--no-open` flag, printed URL.
