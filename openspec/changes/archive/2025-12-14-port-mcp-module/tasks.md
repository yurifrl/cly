# Tasks

## Foundation
- [x] Create `modules/mcp/` directory structure
- [x] Add `cmd.go` with Cobra registration (`Register(parent)`)
- [x] Port `internal/source/mcp.go` types to `modules/mcp/mcp.go`
- [x] Port `internal/source/catalog.go` for loading MCP definitions
- [x] Port `internal/source/parser.go` for YAML/JSONC parsing

## Standalone Binary
- [x] Create `cmd/mcp/main.go` for standalone binary
- [x] Root command uses same module code

## Source Management
- [x] Port `internal/source/add.go` for adding MCPs to catalog
- [x] Port `internal/source/remove.go` for removing MCPs from catalog

## Adapters
- [x] Port `internal/adapters/adapter.go` interface
- [x] Port `internal/adapters/claude.go` (Claude Code `~/.claude.json`)
- [x] Port `internal/adapters/cursor.go` (Cursor `~/.cursor/mcp.json`)
- [x] Port `internal/adapters/desktop.go` (Claude Desktop)

## Config
- [x] Port `internal/config/config.go` global config types
- [x] Port `internal/config/loader.go` for loading config
- [x] Port `internal/config/parser.go` for YAML/JSONC parsing
- [x] Support project config (`.mcp.yaml`)

## Context Detection
- [x] Port `internal/context/context.go` types
- [x] Port `internal/context/detector.go` logic
- [x] Integrate with config file (`defaults.ai`, `defaults.scope`)

## Validation
- [x] Port `internal/validation/validator.go`
- [x] Validation result types and issue tracking

## TUI Model
- [x] Create `modules/mcp/model.go` with Model struct
- [x] Port ListItem types (MCP, Preset, Tag, Separator, Validation)
- [x] Port buildDisplayItems logic
- [x] Port hidden sections state

## TUI Update
- [x] Create `modules/mcp/update.go`
- [x] Port keyboard navigation (up/down/space/enter)
- [x] Port search mode (/ to search, esc to clear)
- [x] Port section expansion (presets, tags)
- [x] Port section collapsing (0-3 keys)
- [x] Port context switcher overlay (c key)
- [x] Port help overlay (? key)

## TUI View
- [x] Create `modules/mcp/view.go`
- [x] Port checkbox rendering with status indicators
- [x] Port status bar with context and pending changes
- [x] Port scroll handling
- [x] Port validation section rendering
- [x] Port context switcher view
- [x] Port help view

## CLI Commands
- [x] `cly mcp` - launch TUI (default)
- [x] `cly mcp apply` - launch TUI explicitly
- [x] `cly mcp list` - list installed MCPs
- [x] `cly mcp list --all` - list all available
- [x] `cly mcp list -l` - detailed view
- [x] `cly mcp switch <name>` - toggle MCP
- [x] `cly mcp switch --on/--off` - force state
- [x] `cly mcp switch -p <preset>` - switch by preset
- [x] `cly mcp switch -t <tag>` - switch by tag
- [x] `cly mcp switch --all` - switch all MCPs
- [x] `cly mcp add <name> <cmd> [args...]` - add MCP to sources
- [x] `cly mcp add` flags: --transport, --env, --header, --tags, --description, --file
- [x] `cly mcp remove <name>` - remove MCP from sources
- [x] `cly mcp context` - show/set default context
- [x] `cly mcp context --show` - show current
- [x] `cly mcp context --which` - show config path
- [x] `cly mcp doctor` - health check
- [x] `cly mcp completion <shell>` - generate completions

## Shell Completions
- [x] Port bash completion generation
- [x] Port zsh completion generation
- [x] Port fish completion generation

## Tests
- [x] Test catalog loading from YAML files
- [x] Test catalog loading from JSONC files
- [x] Test catalog filter functionality
- [x] Test TUI cursor movement
- [x] Test TUI expand/collapse presets
- [x] Test TUI space toggle
- [x] Test TUI search mode
- [x] Test TUI help toggle
- [x] Test validation logic (missing command, command not in PATH)
