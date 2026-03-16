# Changelog

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
