# Proposal: add-bundle-command

## Summary

Add `cly bundle [type]` command for unified declarative package management across brew, go, js, and python ecosystems.

## Motivation

User has separate bundler scripts (`jsbundle`, `gobundle`, `pythonbundle`) in DotFiles. Consolidating into `cly bundle` provides:
- Single entry point for all package management
- Consistent flags and behavior across ecosystems
- Reduced shell script maintenance

## Scope

**In scope:**
- `cly bundle` command with subcommand dispatching
- Support for brew (default), go, js, python types
- Flags: `--edit/-e`, `--no-edit`, `--dry-run`, `--file/-f`
- Bundle file parsing (comments, blanks ignored)
- State file tracking for install/uninstall diff

**Out of scope:**
- TUI interface (this is a shell wrapper utility)
- New bundle file formats (keep existing ~/.config/ files)
- Package version pinning beyond what tools support

## Approach

Non-TUI module following `modules/bundle/` pattern. Each bundler type implemented as separate file sharing common interface. Shells out to underlying tools (brew, bun, go, uv).

## Spec Deltas

- **bundle-management** (new): Requirements for bundle command behavior

## Risk

Low. Wraps existing proven scripts. No changes to other modules.
