# Changelog

## 2026-04-05 Charm Stack Skill Rewrite for v2
- Session ID: 2df1782c-6f24-4bc6-8be4-a3d56292c3c1
- Session File: /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/2026-04-01T12-47-00-611Z_2df1782c-6f24-4bc6-8be4-a3d56292c3c1.jsonl
- Session Name: 2026-04-04-2100-oi-spell-checker
- Context Name: 2026-04-04-2100-oi-spell-checker

### Changed
- `.agents/skills/charm-stack/SKILL.md` — complete rewrite for Charm v2 (released Feb 2026). Covers new import paths (`charm.land/*/v2`), declarative `tea.View` return type, `tea.KeyPressMsg`/`tea.KeyReleaseMsg`, split mouse messages, Lipgloss v2 color model (`LightDark` replaces `AdaptiveColor`), Bubbles v2 functional options and getter/setter API, Huh v2 integration, and v1-to-v2 migration reference table.

## 2026-03-30 Fix gc Git Commits Path Doubling and Continue-on-Error
- Session ID: 1b0f6310-6e22-4b78-b38e-ef6b01d8d629
- Session File: /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/2026-03-30T17-56-30-795Z_1b0f6310-6e22-4b78-b38e-ef6b01d8d629.jsonl
- Session Name: 2026-03-30-1527-pi-tree-tui-tests
- Context Name: 2026-03-30-1527-pi-tree-tui-tests

### Fixed
- `gc` (git-commits) failed with a doubled path prefix when run from a subdirectory of the repo. `gitExec`/`gitOutput`/`gitRawOutput` and `git apply` now set `cmd.Dir` to the repo root via a new `repoRoot()` helper (`git rev-parse --show-toplevel`) in `changeset.go`.

### Changed
- On commit failure, `Execute()` now skips the failing group (unstages partial staging) and continues with remaining commits instead of rolling back everything. `CommitResult` gains `Skipped` and `Err` fields.
- `pipeline.go` prints `✗ SKIP` for failed groups and shows `Done! Created N/M commits.` summary. Error is returned only after all results are printed.
- Added `SilenceUsage: true` to the `git-commits` cobra command so the usage block no longer prints on runtime errors.

## 2026-03-30 Pi-Tree Enter Key Opens Sessions
- Session ID: 49cd5a12-5947-4118-96f8-1d611b7af6fe
- Session File: /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/2026-03-30T13-32-19-031Z_49cd5a12-5947-4118-96f8-1d611b7af6fe.jsonl
- Session Name: 2026-03-30-1521-pi-tree-tui-tests
- Context Name: 2026-03-30-1521-pi-tree-tui-tests

### Fixed
- `modules/pi-tree/tui.go`: Enter key now opens the selected session in an existing or new workspace. Previously the `enter` case only handled history view and did nothing in the normal tree view.

## 2026-03-30 Dotfiles Eval Subcommand

- Session ID: d75e5d8b-bb1c-4cee-b1f2-363e99c975a6
- Session File: /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/2026-03-30T13-30-38-552Z_d75e5d8b-bb1c-4cee-b1f2-363e99c975a6.jsonl
- Session Name: 2026-03-30-1521-pi-tree-tui-tests
- Context Name: 2026-03-30-1521-pi-tree-tui-tests

### Added
- `cly dotfiles eval [src]` — re-applies a single mapping from `dotfiles.conf` by source path (arg or stdin); handles both `.jsonc -> .json` copies and regular symlinks; matches by full path or basename

## 2026-03-27 cmux Integration Package

- Session ID: c3c7de12-52fa-42d4-a8bf-e4134644345a
- Session File: /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/2026-03-27T03-52-19-359Z_c3c7de12-52fa-42d4-a8bf-e4134644345a.jsonl
- Session Name: 2026-03-27-1601-dotfiles-lock-cmux
- Context Name: 2026-03-27-1601-dotfiles-lock-cmux

### Added
- `pkg/cmux` — reusable cmux terminal multiplexer integration package with `Available()`, `Notify(ctx, title, body)`, `SetStatus(ctx, key, value, ...opts)`, `ClearStatus(ctx, key)`; all methods no-op when `CMUX_WORKSPACE_ID` is unset
- `cmux.Notify` call at end of `modules/dotfiles` `runSync` as reference usage — sends "Sync complete" notification when inside cmux

## 2026-03-27 Dotfiles Lock File for Stale Artifact Cleanup
- Session ID: 676b1c93-f202-4e02-ad7e-f0930a52789f
- Session File: /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/2026-03-27T12-52-24-946Z_676b1c93-f202-4e02-ad7e-f0930a52789f.jsonl
- Session Name: 2026-03-27-1601-dotfiles-lock-mux
- Context Name: 2026-03-27-1601-dotfiles-lock-mux

### Added
- `modules/dotfiles/lock.go` — lock file system at `~/.local/share/cly/dotfiles/dotfiles.lock` tracking all applied artifacts (symlinks, jsonc copies, jobs, install commands, op mappings) across runs
- `modules/dotfiles/lock_test.go` — 9 tests for lock build, diff detection, and JSON round-trip
- `RemoveJobByName(name string)` in `jobs.go` — removes a single launchd job by name (plist + script + once-state)
- `RemoveJsoncCopy(m Mapping) bool` in `jsonc.go` — deletes generated JSON destination file
- `RemoveOpMapping(m OpMapping) bool` in `op.go` — deletes injected 1Password destination file
- `--no-it` flag on `cly dotfiles` — skips interactive prompts for non-interactive use

### Changed
- `cly dotfiles` now diffs lock on every run: auto-removes stale symlinks, jsonc copies, jobs, and op mapping files when their entries are deleted from `dotfiles.conf`
- Removed install commands trigger a prominent red interactive banner requiring Enter to continue (user must manually undo; bypassable with `--no-it`)

## 2026-03-12

### Added
- `cly zs` — interactive Zellij smart sessionizer that combines existing sessions, zoxide directories, and optional layout selection

## 2026-03-03

### Added
- `cly agents start` background process model with PID/status files under `~/.config/cly/`
- `cly agents add [repo]` to register repositories in global config (`~/.config/cly/agents.yaml`)
- `cly agents logs` with `--tail` and `--follow` for daemon log inspection

### Changed
- Replaced `cly agents run` with `cly agents sync [repo]` as one-shot sync + exit
- `cly agents` now stores global settings and tracked repos in `~/.config/cly/agents.yaml`
- Daemon sync now runs across configured repos on a periodic loop, enabling newly added repos without restart

### Removed
- `cly agents configure` command
- Legacy socket-based daemon control path

## 2026-02-13

### Added
- `cly agents` module — syncs `.agents/` configs to IDE directories (.claude, .opencode, .crush)
- `cly agents configure` command — bootstraps `agents.yaml` config (`--local` for project-level)
- Bidirectional sync — edits in target dirs (`.claude/`, `.opencode/`) sync back to `.agents/` for non-transformed files
- Daemon mode with fsnotify file watching, debounced sync, and reverse sync for target dirs
- JSONC→JSON transform with env var interpolation and comment stripping
- SKILL.md allowed-tools stripping for OpenCode
- Unix socket daemon with status/stop commands
- `--dry-run`, `--global`, `-i` flags for sync control

### Changed
- Config format switched from `ai.json` (JSONC) to `agents.yaml` (YAML)
- No config = no sync — sync is opt-in per scope (requires `agents.yaml` to exist)
- Simplified global source dirs — removed `~/.config/ai`, only `~/.agents/`

## [1.0.4] - 2025-12-22

- fix: backup commands now create ~/Workdir if it doesn't exist
- chore: reorganize release command to .claude/commands

## [1.0.3] - 2025-12-22

- refactor: centralize dotfiles_dir configuration at app level
- fix: add sudo permissions to install script for system-wide installation
- refactor: improve config resolution in dotfiles and helpy modules
- chore: add release command documentation

## [1.0.2] - Previous Release

Initial stable release with core functionality.
