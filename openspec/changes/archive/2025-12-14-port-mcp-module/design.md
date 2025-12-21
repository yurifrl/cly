# Design: MCP Module Architecture

## Dual Binary Architecture

```
cmd/
├── cly/main.go      # imports modules/mcp, registers as subcommand
└── mcp/main.go      # imports modules/mcp, uses as root command

modules/mcp/
└── ...              # shared code for both binaries
```

**Usage:**
```bash
# As CLY subcommand
cly mcp list --all
cly mcp switch github --on

# As standalone binary
mcp list --all
mcp switch github --on
```

The module exposes `NewRootCmd()` that returns a `*cobra.Command`.
- `cmd/cly/main.go` adds it as subcommand: `rootCmd.AddCommand(mcp.NewRootCmd())`
- `cmd/mcp/main.go` uses it as root: `mcp.NewRootCmd().Execute()`

## Module Structure

```
modules/mcp/
├── cmd.go           # Cobra registration, all subcommands
├── model.go         # Bubbletea Model, state, Init()
├── update.go        # Message handling, Update()
├── view.go          # Rendering, View()
├── help.go          # Help overlay view
├── context_switcher.go # Context switcher overlay
├── operations.go    # Apply changes, save preferences
├── types.go         # MCP, Catalog, ListItem types
├── catalog.go       # Load MCPs from ~/.config/mcpcli/mcps/
├── source.go        # Add/remove MCPs from source files
├── parser.go        # YAML/JSONC parsing
├── adapters.go      # AI tool config read/write (claude, cursor, desktop)
├── context.go       # Context detection (ai:scope)
├── config.go        # Global and project config loading
├── validation.go    # Validation logic and issue tracking
└── completion.go    # Shell completion generation (bash, zsh, fish)
```

## Key Types

```go
// MCP represents an MCP server definition
type MCP struct {
    Name        string
    Command     string
    Args        []string
    Env         map[string]string
    Tags        []string
    Description string
    SourceFile  string
}

// Context represents the target AI tool and scope
type Context struct {
    AI    string // "claude", "cursor"
    Scope string // "user", "project"
}

// Adapter reads/writes config for an AI tool
type Adapter interface {
    ReadConfig(scope string) (*ToolConfig, error)
    WriteConfig(scope string, mcps []MCP) error
    GetConfigPath(scope string) (string, error)
}
```

## State Flow

```
User launches `cly mcp`
       │
       ▼
┌─────────────────┐
│ Load Catalog    │ ~/.config/mcpcli/mcps/*.yaml
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Detect Context  │ config → env → defaults
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Get Adapter     │ claude.go / cursor.go
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Read Installed  │ adapter.ReadConfig()
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Build Model     │ display items, checked state
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Run TUI         │ Bubbletea Program
└─────────────────┘
```

## Style Adaptation

Replace mcpcli's hardcoded colors with `pkg/style/` theme:

```go
// mcpcli (before)
var colorGreen = lipgloss.Color("82")
titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))

// CLY (after)
import "github.com/yurifrl/cly/pkg/style"
checkedStyle := style.Success
titleStyle := style.Title
```

## Config Files

| Path | Purpose |
|------|---------|
| `~/.config/mcpcli/mcps/*.yaml` | MCP source definitions |
| `~/.config/mcpcli/config.yaml` | Defaults (ai, scope) |
| `~/.claude.json` | Claude Code installed MCPs |
| `~/.cursor/mcp.json` | Cursor installed MCPs |

## Full Feature Parity

All mcpcli features are ported:
- Validation section with error/warning display
- Doctor command with `--validate` flag
- Add/Remove commands for source management
- All adapters: Claude Code, Cursor, Claude Desktop
- Context switcher TUI overlay (c key)
- Hidden sections toggle (0-3 keys)
- Shell completions (bash, zsh, fish)
- Presets and tags with expansion
