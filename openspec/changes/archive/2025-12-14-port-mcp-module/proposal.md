# Port MCP Manager as CLY Module

## Summary

Port the mcpcli TUI (from `.references/mcpcli`) as a new CLY module at `modules/mcp/`. The module manages MCP (Model Context Protocol) servers across AI tools (Claude Code, Cursor, Claude Desktop) with a Bubbletea-based TUI.

## Motivation

The mcpcli tool is a standalone CLI that manages MCP server configurations. Porting it to CLY:
- Consolidates utilities into a single tool
- Adapts the TUI to match CLY's visual patterns and styles
- Follows CLY's modular architecture (Cobra registration, self-contained modules)

## Scope

**Full port - all mcpcli features:**
- Core TUI for browsing/toggling MCP servers
- Context detection (which AI tool + scope)
- Adapter pattern for all AI tools (Claude Code, Cursor, Claude Desktop)
- CLI subcommands: `cly mcp` (TUI), `cly mcp list`, `cly mcp switch`
- Source catalog loading from `~/.config/mcpcli/mcps/*.yaml`
- `mcp add` / `mcp remove` commands (source editing)
- `mcp doctor` with validation
- `mcp context` command and TUI context switcher
- `mcp completion` for bash/zsh/fish
- Project-level config (`.mcp.yaml`)
- Section collapsing (0-3 keys)
- Presets and tags with expansion

## Build Targets

The module builds two ways:
- `cly mcp ...` - as subcommand of main binary
- `mcp ...` - standalone binary with same functionality

```
cmd/
├── cly/main.go      # main binary, registers all modules
└── mcp/main.go      # standalone mcp binary
```

## Key Adaptations

| mcpcli | CLY module |
|--------|-----------|
| `flag` package (manual parsing) | Cobra commands with flags |
| Standalone `internal/` packages | Self-contained `modules/mcp/` |
| Custom styles in `view.go` | Shared `pkg/style/` theme |
| `github.com/yurifrl/mcp` imports | Local package paths |

## Risks

- Config file format compatibility: mcpcli uses YAML/JSONC, CLY uses Viper
- UI complexity: mcpcli has sections, search, context switching - may need simplification
- External dependencies: adapters read/write to Claude/Cursor config files

## Related Specs

- `cli-foundation` - Module registration pattern
- `config-management` - Viper integration
