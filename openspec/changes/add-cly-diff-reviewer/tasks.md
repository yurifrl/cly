# Tasks: add-cly-diff-reviewer

## 1. Scaffold

- [ ] 1.1 Create `modules/diff/` with stub `cmd.go` + `Register(parent)`
- [ ] 1.2 Register in `cmd/root.go`
- [ ] 1.3 Add `--port` and `--no-open` flags to cmd
- [ ] 1.4 Verify `cly diff --help` prints

## 2. Git layer

- [ ] 2.1 Write tests for `git.go` (status parse, numstat, hunks, untracked, binary marker)
- [ ] 2.2 Implement `git.go` — `ListChangedFiles()`, `DiffFile(path)`, `ListUntracked()`
- [ ] 2.3 Handle binary marker → set `binary: true`
- [ ] 2.4 Handle empty diff → return `[]File{}`
- [ ] 2.5 Handle not-a-repo → typed error

## 3. Bd layer

- [ ] 3.1 Write tests with fake bd binary on `$PATH` (testdata bash script)
- [ ] 3.2 Implement `bd.go` — `ListLabels()`, `CreateBead(req)`
- [ ] 3.3 Distinguish "bd missing" vs "no db" vs "bd error" (typed errors)
- [ ] 3.4 Parse `--json` output

## 4. HTTP server

- [ ] 4.1 Write handler tests (table-driven, httptest)
- [ ] 4.2 Implement `server.go` — chi router, bind 127.0.0.1, random port
- [ ] 4.3 Implement `handlers.go`:
  - `GET /api/diff`
  - `GET /api/diff/file`
  - `GET /api/labels`
  - `POST /api/bead`
  - `GET /api/health`
- [ ] 4.4 Serve embedded `dist/` at `/`
- [ ] 4.5 SPA fallback — unknown routes serve `index.html`

## 5. Browser launcher

- [ ] 5.1 Implement `browser.go` — macOS/Linux/Windows open
- [ ] 5.2 Honor `--no-open` flag
- [ ] 5.3 Always print URL to stdout before opening

## 6. React frontend scaffold

- [ ] 6.1 `cd modules/diff/web && npm create vite@latest . -- --template react-ts`
- [ ] 6.2 Strip Vite boilerplate, keep only essentials
- [ ] 6.3 Add `.gitignore` for `node_modules/`, `dist/`
- [ ] 6.4 Verify `npm run build` produces `dist/`
- [ ] 6.5 Add `tsconfig.json` strict mode

## 7. Frontend API layer

- [ ] 7.1 `src/types.ts` — mirror Go JSON types
- [ ] 7.2 `src/api.ts` — fetch wrappers for all endpoints
- [ ] 7.3 Error shapes — `{ error: string }`

## 8. Frontend components

- [ ] 8.1 `App.tsx` — layout shell, health check on mount
- [ ] 8.2 `FileList.tsx` — changed-files sidebar
- [ ] 8.3 `DiffView.tsx` — render hunks with +/- coloring
- [ ] 8.4 `EmptyState.tsx` — "nothing to review"
- [ ] 8.5 `BeadModal.tsx` — form (title, desc, type, priority, context, labels)
- [ ] 8.6 `ChipsInput.tsx` — labels chip input w/ suggest + arrow nav
- [ ] 8.7 `ContextPicker.tsx` — file dropdown w/ search + arrow nav
- [ ] 8.8 `hooks/useKeyboard.ts` — global shortcuts (n, esc, ctrl+enter)
- [ ] 8.9 Toast for bead created
- [ ] 8.10 Banner when `health.beadsDb === false`

## 9. Go embed + build

- [ ] 9.1 `//go:embed web/dist/*` in `embed.go`
- [ ] 9.2 `embed_test.go` — asserts dist contains `index.html`
- [ ] 9.3 Add `task build:web` to `Taskfile.yml`
- [ ] 9.4 Make `task build` depend on `build:web`
- [ ] 9.5 CI runs `task build`, verifies embedded assets present

## 10. Integration + manual verify

- [ ] 10.1 Integration test — spawn server, hit `/api/diff`, assert shape
- [ ] 10.2 Integration test — POST bead with fake bd, assert bd called with right args
- [ ] 10.3 Manual: `go run . diff` in cly repo, open browser, see diff
- [ ] 10.4 Manual: create bead, verify `bd-XX` in `bd list`
- [ ] 10.5 Manual: test with bd missing → banner shows
- [ ] 10.6 Manual: test with no .beads/ → banner shows
- [ ] 10.7 Manual: test with empty diff → empty state shows
- [ ] 10.8 Manual: test on linux (if available) via SSH

## 11. Docs

- [ ] 11.1 Update `AGENTS.md` — add `modules/diff/` to structure map
- [ ] 11.2 Add Node 20+ prerequisite to build docs
- [ ] 11.3 README section for `cly diff`
- [ ] 11.4 Move tech-debt items from AGENTS.md that overlap (none yet)

## 12. Release

- [ ] 12.1 Bump `VERSION`
- [ ] 12.2 Commit, push
- [ ] 12.3 Verify CI green, release published
