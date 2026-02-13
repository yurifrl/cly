# Changelog

## 2026-02-13

### Added
- `cly agents` module — syncs `.agents/` configs to IDE directories (.claude, .opencode, .crush)
- Daemon mode with fsnotify file watching and debounced sync
- JSONC→JSON transform with env var interpolation and comment stripping
- SKILL.md allowed-tools stripping for OpenCode
- Unix socket daemon with status/stop commands
- `--dry-run`, `--global`, `-i` flags for sync control

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
