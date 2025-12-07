# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NSX CLI is a Go-based command-line interface tool for NSX team members, built with a modular architecture that supports team-specific commands while maintaining consistency through shared utilities.

## Key Commands

### Development Commands
- `make setup` - Set up the development environment
- `make build` - Build the application
- `make test` - Run all tests (unit tests, coverage, and package coverage)
- `make unit` - Run unit tests
- `make lint` - Run linting for Go and YAML files
- `make clean` - Clean generated files
- `go run main.go` - Run the CLI locally during development
- `go install .` - Install the binary locally

### Testing Commands
- `make unit` - Run unit tests with testdox format and coverage
- `make coverage` - Generate and display test coverage report
- `make coverage-html` - Generate HTML coverage report and open it
- `make update-unit-snapshots` - Update test snapshots

### Linting Commands
- `make lint-golang` - Lint Go files using golangci-lint
- `make lint-yaml` - Lint YAML files using yamllint

## Workflow Reminders
- Before commit and push, always run make lint, make test, make gen, make format

## Architecture

### Core Structure
- **main.go**: Entry point that calls `cmd.Execute(version)`
- **cmd/**: Contains command definitions using Cobra framework
  - `cmd/root.go`: Root command setup with global flags (`--skin`, `--debug`)
- **shared/**: Reusable utilities across all commands
  - `shared/config/`: Configuration management with encryption support
  - `shared/interact/`: User interaction utilities (Info, Error, Debug messages)
  - `shared/skin/`: Skin management system for CLI themes
  - `shared/style/`: Consistent styling system using Lipgloss for colors, themes, and formatting
  - `shared/ui/`: Reusable UI components for terminal interfaces (tables, forms, etc.)
  - `shared/googlex/`: Google API integration utilities
- **team/**: Team-specific command modules
  - Each team has its own subdirectory (e.g., `team/customer/`)
  - Teams register their commands in `cmd/root.go`

### Configuration System
- Uses TOML format for configuration files
- Supports encryption for sensitive data using AES-GCM
- Configuration stored in `~/.config/nsx/` (configurable via `NSX_CONFIG_PATH`)
- Generic `config.Load[T]()` function for type-safe config loading

### Team Command Structure
Teams should implement:
- `cmd/root.go`: Team's root command
- `internal/`: Team-specific internal packages
  - `internal/database/`: Database connection utilities
  - `internal/views/`: UI view components for displaying data
- `customercfg/`: Team-specific configuration types

### Key Dependencies
- **Cobra**: CLI framework for command structure
- **Charmbracelet**: TUI components (bubbles, bubbletea, lipgloss)
- **Google APIs**: OAuth2 and Sheets integration
- **TOML**: Configuration file format
- **MySQL**: Database connectivity

## Development Patterns

### Adding New Commands
1. Create command file in `cmd/` directory
2. Use Cobra's `cobra.Command` struct
3. Register command in `init()` function with `rootCmd.AddCommand()`
4. Leverage shared utilities from `shared/` package

### Adding Team Commands
1. Create team directory under `team/`
2. Implement team's root command
3. Register team commands in `cmd/root.go`
4. Use shared utilities for consistency

### Configuration Usage
```go
// Define config struct with TOML tags
type Config struct {
    Value string `toml:"value"`
}

// Load configuration
cfg, err := config.Load[Config]("filename.toml")
```

### User Interaction
```go
import "github.com/NSXBet/nsx-cli/shared/interact"

interact.Info("Information message")
interact.Error("Error message")
interact.Debug("Debug message") // Only shown with --debug flag
```

### UI Components
```go
import "github.com/NSXBet/nsx-cli/shared/ui"

// Create a table with generic type support
options := ui.TableOption[MyDataType]{
    Title: "My Table",
    Columns: []table.Column{
        table.NewColumn("id", "ID", 5),
        table.NewFlexColumn("name", "Name", 2),
    },
    Data: myData,
    RowFunc: func(item MyDataType) table.Row {
        return table.NewRow(table.RowData{
            "id": item.ID,
            "name": item.Name,
        })
    },
    Footer: "Use ↑/↓ to navigate • Press q to quit",
}

// Run the table as a standalone program
ui.RunTable(options)
```

## Environment Variables
- `NSX_CONFIG_PATH`: Override default config directory
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret

## Build System
- Uses Go modules with Go 1.24.0
- Makefile-based build system with modular makefiles in `makefiles/`
- Version injection at build time
- Cross-platform support (macOS, Linux, Windows)

## Testing
- Uses `gotestsum` for enhanced test output
- Coverage reporting with `gocovmerge`
- Package-level coverage analysis
- Test format: testdox for readable output