# CLY Agent Instructions

Modular Go CLI for utilities and TUI demos, built with Cobra + Charm libraries.

## Commands

```bash
# Development
task build          # Build cly binary to dist/
task build:mcp      # Build mcp binary to dist/
task test           # Run all tests
task run -- <args>  # Run without building (go run . <args>)

# Installation
task install        # Build and install cly to /usr/local/bin
task install:mcp    # Build and install mcp to /usr/local/bin

# Run directly
go run . uuid                    # UUID generator
go run . demo spinner            # Spinner demo
go run . mcp list                # MCP server manager
go run ./cmd/mcp --help          # Standalone mcp binary
```

## Project Structure

```
cmd/root.go              # Root command, module registration
main.go                  # Entry point
modules/                 # Self-contained command modules
  uuid/                  # Simple module (cmd.go + uuid.go)
  mcp/                   # Complex module (many files)
  demo/                  # Parent with 48 subcommand modules
  scraper/               # Multi-package module (cmd/, browser/, aliexpress/)
pkg/                     # Shared utilities
  config/                # Viper-based config (YAML + env vars)
  store/                 # SQLite key-value storage
  style/                 # Lipgloss theme
  session/               # Session ID generator
  notify/                # Desktop/Zellij notifications
```

## Adding Modules

Modules self-register. Two patterns:

**Simple module** (single command):
```go
// modules/mymodule/cmd.go
package mymodule

import "github.com/spf13/cobra"

func Register(parent *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "mymodule",
        Short: "Description",
        RunE:  run,
    }
    parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

**Parent with subcommands** (like demo/):
```go
// modules/mymodule/cmd.go
package mymodule

import (
    "github.com/spf13/cobra"
    "github.com/yurifrl/cly/modules/mymodule/sub1"
)

var Cmd = &cobra.Command{
    Use:   "mymodule",
    Short: "Parent command",
}

func Register(parent *cobra.Command) {
    parent.AddCommand(Cmd)
}

func init() {
    sub1.Register(Cmd)
}
```

**Register in root** - add one line to `cmd/root.go` init():
```go
mymodule.Register(RootCmd)
```

## Code Patterns

**Always use RunE** (not Run) - returns errors properly:
```go
RunE: func(cmd *cobra.Command, args []string) error {
    return doWork()
}
```

**TUI commands use Bubbletea**:
```go
func run(cmd *cobra.Command, args []string) error {
    p := tea.NewProgram(initialModel())
    _, err := p.Run()
    return err
}
```

**Config access** (Viper):
```go
import "github.com/yurifrl/cly/pkg/config"

cfg, _ := config.Load()
value := config.GetString("modules.mymodule.setting")
```

**Shared styles** (Lipgloss):
```go
import "github.com/yurifrl/cly/pkg/style"

fmt.Println(style.TitleStyle.Render("Title"))
fmt.Println(style.GreenStyle.Render("Success"))
```

## Testing

Integration tests preferred over mocks. Use real dependencies.

```bash
go test ./...                          # All tests
go test ./modules/mcp/... -v           # Specific module
go test ./pkg/config/... -run TestLoad # Specific test
```

Test patterns:
- `*_test.go` next to source files
- Use `testify/assert` and `testify/require`
- `testdata/` for fixtures (e.g., `modules/update/testdata/`)

## Configuration

Config files searched in order:
- `~/.config/cly/config.local.yaml` (not committed)
- `~/.config/cly/config.yaml`
- `modules/config/config.yaml` (defaults)

Environment variables: `CLY_` prefix (e.g., `CLY_APP_DEBUG=true`)

Supports 1Password secrets: `op://Personal/item/field` references resolved at load time.

## Two Binaries

The repo produces two binaries:

- **cly** - Main CLI with all modules (`main.go`)
- **mcp** - Standalone MCP manager (`cmd/mcp/main.go`)

Both share `modules/mcp/` code. The mcp binary is for use as an MCP server itself.

## Release Process

Version is in `VERSION` file. CI creates GitHub releases automatically:

- Bump `VERSION`
- Push to main
- CI builds darwin-amd64 and darwin-arm64 binaries
- Creates GitHub release with checksums

PR checks verify VERSION hasn't been used as a tag yet.

## Spec-Driven Development

For non-trivial changes, use OpenSpec workflow. Read `openspec/AGENTS.md` when:
- Planning new capabilities
- Making breaking changes  
- Architecture changes

## Key Dependencies

- **Cobra** - CLI framework
- **Viper** - Configuration
- **Bubbletea** - TUI framework (Elm architecture)
- **Bubbles** - TUI components (spinner, list, table, etc.)
- **Lipgloss** - Terminal styling
- **chromedp** - Browser automation (scraper module)

## Gotchas

- Modules are self-contained: no cross-module imports except through pkg/
- Demo modules are adapted from official Bubbletea examples in `.references/`
- Tests skip if dependencies unavailable (e.g., 1Password CLI)
- Config falls back to embedded defaults if no file exists

## Tech Debt: Helpy AI Chat & LLM

### Critical — Fix first
- [ ] **AI chat is completely broken**: responses trickle in one word at a time with multi-second gaps between tokens. Should be instantaneous — other AI tools stream thousands of context tokens faster. The TUI locks completely during streaming: can't cancel, can't scroll, can't quit. Unusable in current state.
- [ ] **No doc context passed**: the AI system prompt should include the full current document + frontmatter, but the context isn't reaching the model reliably. This is the core value prop — the AI must know what you're reading.
- [ ] **`llm-chat` is broken**: still references the old mods-based client patterns internally. Needs full refactor to use `pkg/llm` SDK directly (Anthropic/OpenAI).

### Aliases
- [ ] Add `lchat` and `lc` as aliases for `llm-chat` (same pattern as `hy` for `helpy`).

### Features
- [ ] **Pipe support (`-p`)**: `echo "what is this?" | cly helpy -p` should accept stdin and print the AI answer inline (no TUI), like a one-shot query with doc context.
- [ ] **Be AI-agnostic**: remove all mentions of "Claude" from `--help` text, flag descriptions, and function names. The tool supports Anthropic and OpenAI — language should reflect that.

### UX
- [ ] Show a visible streaming indicator (spinner + token count or elapsed time) so it's clear the AI is still working.
- [ ] Allow typing the next question while a response is still streaming.
