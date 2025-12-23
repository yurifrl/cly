---
triggers: [module, what does, list modules, find module]
---

# Module Map

## Utilities

**uuid** - Generate UUIDs (v4/v7/multiple) | modules/uuid/
**backup** - GCS backup/restore ~/Workdir | modules/backup/
**bundle** - Package manager (brew/go/js/python) | modules/bundle/
**scraper** - AliExpress product scraper (chromedp) | modules/scraper/
**mcp** - MCP server manager (Claude/Cursor/Desktop) | modules/mcp/
**dotfiles** - Symlink manager (declarative) | modules/dotfiles/
**helpy** - Markdown viewer with search | modules/helpy/
**ai** - Chat via mods CLI | modules/ai/
**claude** - Claude Code wrapper + sessions | modules/claude/
**notify** - Claude Code hooks, notifications | modules/notify/
**update** - Self-updater (GitHub releases) | modules/update/
**config** - Config management commands | modules/config/

## Demo (48 Bubbletea Examples)

All in modules/demo/* - See .ai/skills/charm-stack/SKILL.md

## Infrastructure

**pkg/config** - Viper config + 1Password secrets
**pkg/store** - SQLite state tracking
**pkg/style** - Lipgloss themes
**pkg/session** - Session naming + Zellij
**pkg/notify** - Multi-notifier (beeep + Zellij)
