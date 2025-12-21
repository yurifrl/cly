# Proposal: Add GitHub Release Download

## Summary

Implement `zellij_plugin` function for dotfiles install commands, downloading `.wasm` files from GitHub releases.

## Motivation

Current `dotfiles.conf` uses `!zellij_plugin https://github.com/user/repo` to download Zellij plugins. The Go implementation needs to be implemented.

## Scope

**In Scope:**
- `zellij_plugin <github_url>` function
- Download from `/releases/latest/download/<repo>.wasm`
- Save to configurable directory (`modules.dotfiles.zellij_plugins_dir`)
- Default to `~/.config/zellij/plugins/`

**Out of Scope:**
- GitHub API authentication (public releases only)
- Asset pattern matching (convention: `<repo>.wasm`)

## Design

Simple direct download using `net/http`:
1. Parse `github.com/owner/repo` from URL
2. Construct: `https://github.com/owner/repo/releases/latest/download/repo.wasm`
3. HTTP GET with redirect following
4. Write to `~/.config/zellij/plugins/repo.wasm`

No GitHub API needed - `/releases/latest/download/` is a direct download endpoint.
