# add-skills-pi-ext-install - Implementation Tasks

## Phase 1: Foundation
- [x] 1.1 Create `modules/skills/` directory with `cmd.go` skeleton (Register + parent `skills` cobra command)
- [x] 1.2 Create `modules/skills/embedded/agents-session/SKILL.md` with initial session-rules content (best-effort guidance for calling `cly agent-session save` / `cly gs save`)
- [x] 1.3 Create `modules/pi/` directory with `cmd.go` skeleton (Register + parent `pi` + child `extensions` cobra commands)
- [x] 1.4 Register both modules in `cmd/root.go` init()

## Phase 2: Core Implementation
- [x] 2.1 Implement embedded FS walker in `pkg/embedfs/install.go` (`Install` + `InstallSelected` + `ResolveTarget`). Lives in `pkg/` because it is generic infrastructure, not skill-specific — both `modules/skills` and `modules/pi` consume it.
- [x] 2.2 Implement `cly skills install [name...]` subcommand: `--target` (default `~/.agents/skills/`), `--dry-run`; cherry-pick by positional names (default: all); overwrite by default; verbose stdout.
- [x] 2.3 Implement `cly pi extensions install` subcommand: `--target` (default `~/.pi/agent/extensions/`), `--dry-run`; installs all embedded extensions (per decision: extensions are all-or-nothing, users can toggle off in pi).
- [x] 2.4 Author `modules/pi/embedded/pi-cly/` pi extension: single extension bundle (`pi-cly`) with `package.json` + `index.ts`. Decision: one `pi-cly` extension containing the `/save` command, not a per-command folder.
- [x] 2.5 Implement `/save` argument parser (`parseSaveArgs`): kv pairs (`key="value"` / `key=value`) extracted first, remaining positional text → `name`. Only `description` is honored as a kv key today.
- [x] 2.6 Implement `/save` invocation of `cly as save <id> <name> <description>` via `spawn("cly", ...)` — using the existing `as` alias of `agent-session`. No new aliases added.

## Phase 3: Testing & Documentation
- [x] 3.1 Integration tests for `cly skills install`: write, overwrite, dry-run, cherry-pick, unknown-name error, stdout assertions.
- [x] 3.2 Integration tests for `cly pi extensions install`: write + dry-run against `t.TempDir()`.
- [x] 3.3 Update `README.md` with the two new commands and example `dotfiles.conf` snippet.
- [x] 3.4 Manual test plan documented in `modules/pi/embedded/pi-cly/README.md`.

## Phase 4: Validation
- [x] 4.1 `go test ./modules/skills/... ./modules/pi/...` — 7 tests pass.
- [x] 4.2 `task build` — binary builds with embedded assets.
- [x] 4.3 Verified `cly skills install --dry-run --target /tmp/...` prints `would write .../agents-session/SKILL.md`.
- [x] 4.4 Verified `cly pi extensions install --dry-run --target /tmp/...` prints `would write .../pi-cly/{index.ts,package.json}`. Live-in-pi manual test left to the user per the manual test plan.
- [x] 4.5 Ready for merge.

**Notes:**
- Decision recap (from user during implementation): skills cherry-pickable, extensions all-or-nothing (toggle off in pi), single `pi-cly` extension instead of per-command extensions, install helper lives in `pkg/embedfs` (not inside `modules/skills`), use existing `as` alias (no new `gs` alias).
- `modules/pi/install.go` and `modules/skills/install.go` both call `pkg/embedfs` — no cross-module coupling.
