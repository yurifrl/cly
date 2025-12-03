# CLY - Charm Library Showcase CLI

## Overview

CLY is a modular CLI application built with Go that serves as both a functional tool for managing custom scripts and a comprehensive demonstration of the Charm ecosystem libraries. The project emphasizes clean architecture, extensibility, and rich terminal user interfaces.

## Technology Stack

### Core Framework
- **Cobra** - CLI application framework
  - Command/subcommand structure
  - Argument parsing and validation
  - Auto-generated help and documentation
  - Shell completion (bash, zsh, fish, PowerShell)

### Configuration Management
- **Viper** - Configuration handling
  - YAML-based configuration files
  - Environment variable support
  - Multiple config source precedence
  - Live config reloading support

### Terminal UI Libraries
- **Bubbletea** - TUI framework foundation
  - Event-driven architecture (Elm-inspired)
  - State management
  - Keyboard/mouse input handling
  - Cross-platform rendering

- **Huh** - Interactive forms and prompts
  - Input fields (single/multi-line)
  - Select/MultiSelect components
  - Confirm dialogs
  - Dynamic form fields
  - Built-in validation

- **Lipgloss** - Styling and layout
  - True color support (24-bit)
  - CSS-like styling API
  - Borders, padding, margins
  - Text alignment and formatting
  - Adaptive color schemes

- **Bubbles** - Pre-built TUI components
  - Lists with filtering
  - Tables
  - Spinners and progress bars
  - Text input/textarea
  - Viewport (scrolling)
  - File picker
  - Timer/stopwatch
  - Help system

## Architecture

### Modularity Pattern

The CLI follows a modular architecture inspired by `~/Workdir/Nsx/nsx-cli/`:

```
cly/
├── cmd/
│   ├── root.go           # Root command and global flags
│   ├── demo/             # Demo command module
│   │   ├── demo.go
│   │   └── tui.go
│   └── script/           # Script management module
│       ├── script.go
│       ├── add.go
│       ├── list.go
│       └── run.go
├── internal/
│   ├── config/           # Viper configuration management
│   ├── tui/              # Reusable TUI components
│   │   ├── theme.go
│   │   ├── forms.go
│   │   └── components.go
│   └── script/           # Script management logic
├── scripts/              # User scripts directory
└── config.yaml           # Default configuration
```

**Key Principles:**
- Each command is a self-contained module
- New commands can be added without modifying core code
- Shared TUI components in `internal/tui/`
- Clear separation between CLI and TUI layers

### Configuration Structure

```yaml
# config.yaml
scripts:
  directory: "./scripts"
  extensions: [".sh", ".py", ".js"]

ui:
  theme: "charm"  # charm, dracula, catppuccin, base16, default
  color_mode: "auto"  # auto, light, dark

editor:
  default: "nvim"
```

## Feature Specifications

### Phase 1: Charm Library Demonstration

A comprehensive `demo` command showcasing all Charm libraries:

```bash
cly demo
```

**Components to demonstrate:**

1. **Bubbletea App Structure**
   - Full-screen TUI with navigation
   - Multiple views/screens
   - Event handling examples

2. **Huh Forms Showcase**
   - All five field types (Input, Text, Select, MultiSelect, Confirm)
   - Validation examples
   - Dynamic forms (fields changing based on input)
   - Theme switcher

3. **Lipgloss Styling Gallery**
   - Color palette showcase (ANSI, 256, TrueColor)
   - Border styles
   - Layout examples (padding, margins, alignment)
   - Adaptive color demonstration

4. **Bubbles Component Library**
   - Interactive list with filtering
   - Data table navigation
   - Progress bar animations
   - Spinner variations
   - Text input/textarea demo
   - File picker
   - Timer/stopwatch
   - Help system

**Interactive Demo Flow:**
```
┌─────────────────────────────────────┐
│  CLY - Charm Library Showcase      │
├─────────────────────────────────────┤
│  → Forms & Input                    │
│    Lists & Tables                   │
│    Styling Gallery                  │
│    Progress & Spinners              │
│    File System                      │
│    Exit                             │
└─────────────────────────────────────┘
```

### Phase 2: Script Management

Commands for managing and running custom scripts:

#### Add Script
```bash
cly script add
```
- Interactive form (Huh) to input script details
- File picker (Bubbles) for script selection
- Metadata: name, description, tags, environment

#### List Scripts
```bash
cly script list [--tag tag1,tag2]
```
- Filterable table (Bubbles) of scripts
- Columns: Name, Description, Tags, Last Run
- Interactive selection to view details or run

#### Run Script
```bash
cly script run <name>
```
- Confirm dialog (Huh)
- Live output with scrolling viewport (Bubbles)
- Progress indicator during execution
- Exit code and duration display

#### Edit Script Metadata
```bash
cly script edit <name>
```
- Pre-filled form (Huh) with existing metadata
- Update without modifying script file

### Phase 3: Advanced Features

- **Script Templates**: Pre-configured script scaffolding
- **Environment Variables**: Per-script environment management
- **Script Chains**: Run multiple scripts in sequence
- **Scheduling**: Cron-like scheduling (view only, execution via system cron)
- **Export/Import**: Share script configurations

## TUI Design Guidelines

### Theme System
- Default to Charm theme
- User-configurable via config.yaml
- Consistent styling across all views
- Respect terminal color capabilities

### Interaction Patterns
- Vim-style keybindings where applicable (j/k navigation)
- Standard shortcuts (Ctrl+C to exit, ? for help)
- Mouse support in compatible terminals
- Progressive disclosure (simple by default, advanced on demand)

### Accessibility
- Huh accessibility mode for screen readers
- High contrast color options
- Keyboard-only navigation
- Clear focus indicators

## Development Roadmap

### Milestone 1: Foundation
- [ ] Project setup (Go modules, directory structure)
- [ ] Cobra root command and basic CLI structure
- [ ] Viper configuration loading from YAML
- [ ] Basic `demo` command skeleton

### Milestone 2: Demo Implementation
- [ ] Bubbletea full-screen app framework
- [ ] Huh forms showcase (all field types)
- [ ] Lipgloss styling gallery
- [ ] Bubbles component demonstrations
- [ ] Navigation between demo sections

### Milestone 3: Script Management (CLI)
- [ ] `script add` command with validation
- [ ] `script list` command with filtering
- [ ] `script run` command with output capture
- [ ] Script metadata storage (YAML)

### Milestone 4: Script Management (TUI)
- [ ] Interactive script list (Bubbles table)
- [ ] Form-based script addition (Huh)
- [ ] File picker integration
- [ ] Live script execution view

### Milestone 5: Polish & Documentation
- [ ] Comprehensive README
- [ ] Example scripts included
- [ ] Shell completion scripts
- [ ] CI/CD setup

## Technical Considerations

### Error Handling
- Graceful degradation for unsupported terminals
- Clear error messages with context
- Recovery suggestions (similar to Cobra's smart suggestions)

### Performance
- Lazy loading of scripts list
- Efficient rendering (Bubbletea's framerate management)
- Config caching

### Testing Strategy
- Unit tests for business logic
- Integration tests for Cobra commands
- Manual TUI testing (automated TUI testing is complex)
- Example scripts for demo purposes

## Success Criteria

1. **Complete Charm Showcase**: All four libraries used extensively
2. **Modular Architecture**: Easy to add new commands/features
3. **Production Quality**: Polished UX, comprehensive error handling
4. **Documentation**: Clear examples and usage instructions
5. **Extensibility**: Other developers can easily add commands

## References

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Huh](https://github.com/charmbracelet/huh) - Forms library
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration
- [NSX CLI](~/Workdir/Nsx/nsx-cli/) - Modularity inspiration
