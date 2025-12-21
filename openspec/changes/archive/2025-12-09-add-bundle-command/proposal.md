# Proposal: add-bundle-command

## Summary

Add `cly bundle [type]` command for unified declarative package management across brew, go, js, and python ecosystems with DuckDB-backed state tracking.

## Motivation

User has separate bundler scripts in DotFiles. Consolidating into `cly bundle` provides:
- Single entry point for all package management
- Cleanup as default behavior (bundle file is source of truth)
- Shared state infrastructure via `pkg/store` (DuckDB)

## Scope

**In scope:**
- `cly bundle [type]` — install + cleanup (default: brew)
- `cly bundle check [type]` — show diff, no changes
- `cly bundle cleanup [type]` — remove unlisted only
- Flags: `--file/-f`, `--verbose/-v`
- DuckDB store in `pkg/store` for go/js/python state
- Dependency injection pattern for Store

**Out of scope:**
- TUI interface (batch operations)
- New bundle file formats (keep existing ~/.config/ files)
- Version pinning beyond tool support

## Approach

Two components:
1. `pkg/store` — generic namespace/key Store interface with DuckDB implementation
2. `modules/bundle` — bundler implementations consuming Store

Brew delegates to `brew bundle` (no Store needed). Go/js/python use Store for tracking.

## Spec Deltas

- **store-infrastructure** (new): Generic key-value store for app state
- **bundle-management** (modified): Updated for subcommands and DuckDB

## Risk

Low. Wraps existing proven scripts. Store is simple key-value pattern.
