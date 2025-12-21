# Project Context

## Purpose
CLY is a modular Go CLI tool for day-to-day utility tasks with beautiful TUI interfaces. It showcases production-ready usage of Charm libraries (Bubbletea, Bubbles, Huh, Lipgloss) while providing genuinely useful utilities.

**Core Goals**:
- Real utilities people actually use (UUID generation, JSON formatting, HTTP requests, encoding/decoding)
- Beautiful TUI interfaces powered by Charm libraries
- Working reference implementations when you need to add TUI features to other projects
- Modular architecture that scales from 5 to 100+ commands

## Tech Stack

### Core Framework
- **Go 1.24** - Primary language
- **Cobra** - CLI framework (commands, flags, help)
- **Viper** - Configuration management (YAML, env vars)

### TUI Libraries (Charm)
- **Bubbletea** - TUI framework (Elm Architecture: Model/View/Update)
- **Bubbles** - Pre-built TUI components (spinner, table, list, progress, etc.)
- **Huh** - Forms and prompts
- **Lipgloss** - Styling (colors, borders, layout)

### Development Tools
- **mise** - Tool version management (Go 1.24)

## Project Conventions

### Code Style
- **Standard Go formatting** (`gofmt`, `goimports`)
- **Package naming**: lowercase, single word, matches directory name
- **File organization**: Small, focused files (prefer `model.go`, `view.go`, `update.go` over monolithic files)
- **Error handling**: Explicit error returns, no panic in library code
- **Naming conventions**:
  - Exported types/functions: PascalCase
  - Unexported: camelCase
  - Interfaces: noun or adjective (Reader, Runnable)

### Architecture Patterns

**1. Command Registration (Query Command Pattern)**
- Single registration point in `cmd/root.go`
- Each module exports `Register(parent *cobra.Command)` function
- Zero coupling between modules

**2. Module Self-Containment**
- Each module lives in `modules/<name>/`
- Contains all code: cmd.go, model.go, view.go, update.go
- No dependencies on other modules

**3. Shared Utilities (Package-Oriented Design)**
- `pkg/config/` - Viper-based configuration
- `pkg/style/` - Lipgloss theme/styles
- `pkg/ui/` - Reusable UI components
- Modules import shared packages, never each other

**4. Bubbletea MVC Pattern**
- `model.go` - State and Init()
- `view.go` - Rendering logic (View())
- `update.go` - Message handling (Update())
- `cmd.go` - Cobra command registration

### Testing Strategy
- **TDD Required** - Write tests first, no code without tests
- Unit tests for each module in `<module>_test.go`
- Table-driven tests for CLI flags and commands
- Integration tests for full command execution
- Test coverage expected in all new code

### Git Workflow
- **Branch naming**: `yurifrl/<type>/<description>`
  - Examples: `yurifrl/feat/add-uuid-command`, `yurifrl/fix/spinner-crash`
- **Commit conventions**: Conventional Commits format
  - `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- **Never push to main directly** - Always use branches
- **Never merge without approval** - PR-based workflow

## Domain Context

### Charm Libraries Ecosystem
CLY is a reference implementation showing how to structure production CLI apps using:
- **Elm Architecture** (Bubbletea's Model/View/Update pattern)
- **Component composition** (Bubbles pre-built components)
- **Declarative styling** (Lipgloss functional styles)

### Modularity Principles
Architecture follows production CLI best practices:
- Command registration pattern for zero-coupling modules
- Module locality and self-containment
- Single binary output with no runtime dependencies

### Reference Examples
Working examples in `references/bubbletea/examples/` provide starting points:
- Don't reinvent - adapt and integrate working code
- Each module maps to a reference example
- Maintain simplicity of original examples while adding modularity

## Important Constraints

### Technical
- **Single binary output** - No external runtime dependencies
- **Cross-platform** - Must work on macOS, Linux, Windows
- **Fast startup** - CLI tools must feel instant (<100ms)
- **Terminal compatibility** - Support standard terminals (no iTerm-specific features)

### Design
- **No "demo" parent command** - Each component is a direct command (`cly spinner`, not `cly demo spinner`)
- **No inter-module dependencies** - Modules only depend on shared packages
- **Configuration locality** - Module-specific config in YAML sections

### User Experience
- **Consistent theming** - All modules use shared Lipgloss styles
- **Keyboard-driven** - All TUIs support standard keybindings (q to quit, arrows to navigate)
- **Helpful defaults** - Commands work without flags, flags provide customization

## External Dependencies

### Required at Runtime
- None (static binary)

### Development References
- `references/bubbletea/` - Official Charm Bubbletea examples
- `references/soft-serve/` - Advanced Charm library usage

### Configuration Files
- `config/config.yaml` - Default configuration (embedded in binary)
- `~/.config/cly/config.yaml` - User overrides
- Environment variables: `CLY_*` prefix (e.g., `CLY_APP_DEBUG=true`)
