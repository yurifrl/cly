# Changelog

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
