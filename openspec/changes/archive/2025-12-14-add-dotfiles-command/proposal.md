# Proposal: Add Dotfiles Command

## Summary

Add `cly dotfiles` command to manage dotfile symlinks from a declarative config file, replacing the existing bash script in `~/DotFiles`.

## Motivation

Currently dotfiles are managed by a ~190 line bash script. Bringing this into cly:
- Consolidates tooling into single binary
- Adds type safety and better error handling
- Enables future TUI enhancements (interactive status, conflict resolution)
- Follows cly's modular architecture patterns

## Scope

**In Scope:**
- Config file parsing (`dotfiles.conf` format)
- Symlink creation/replacement
- Install commands (`!` prefixed lines) with `-i` flag
- Status display showing link states
- Unlink subcommand to remove managed symlinks
- Generic `github_release_download` function (replacing `zellij_plugin`)

**Out of Scope:**
- Backup functionality (errors on conflicts instead)
- Diff/history tracking
- TUI interactive mode (future enhancement)

## User Impact

- `cly dotfiles` - sync symlinks from config
- `cly dotfiles -i` - sync + run install commands
- `cly dotfiles status` - show current state
- `cly dotfiles unlink` - remove all managed symlinks

## Dependencies

- Requires `config-management` spec (for default directory configuration)
