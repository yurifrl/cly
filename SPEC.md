# CLY - Charm CLI Tech Demo

## Overview

CLY is a modular Go CLI application serving as a **tech demo for Charm libraries**. It demonstrates the full capabilities of the Charm ecosystem while providing a framework for adding random scripts/utilities.

## Technology Stack

| Library | Purpose | Version |
|---------|---------|---------|
| **Cobra** | CLI framework - commands, subcommands, args, help | spf13/cobra |
| **Viper** | Config management - YAML files, env vars, flags binding | spf13/viper |
| **Bubbletea** | TUI framework - Elm Architecture (Model-Update-View) | charmbracelet/bubbletea |
| **Bubbles** | Pre-built TUI components | charmbracelet/bubbles |
| **Huh** | Forms and prompts | charmbracelet/huh |
| **Lipgloss** | Styling - colors, borders, layout | charmbracelet/lipgloss |

## Architecture

### Modular Design (inspired by nsx-cli)

```
cly/
├── main.go                 # Entry point
├── cmd/
│   └── root.go             # Root command, registers all modules
├── config/
│   ├── config.go           # Viper setup, YAML config loading
│   └── config.yaml         # Default config template
├── internal/
│   └── style/              # Shared lipgloss styles
│       └── style.go
├── modules/                # Each module is self-contained
│   ├── demo/               # Charm tech demo module
│   │   ├── cmd.go          # Cobra command registration
│   │   ├── model.go        # Bubbletea model
│   │   ├── view.go         # Bubbletea view
│   │   └── update.go       # Bubbletea update
│   └── <future>/           # Future modules follow same pattern
└── pkg/                    # Shared utilities
    └── ui/
        └── components.go   # Reusable Bubbles wrappers
```

### Module Interface

Each module:
- Has **1 entry point** (registers its Cobra command)
- Receives dependencies via **parameters or interfaces**
- Can be **easily extracted** into separate package
- No modification to core code when adding new modules

```go
// modules/demo/cmd.go
func Register(parent *cobra.Command, cfg *config.Config) {
    cmd := &cobra.Command{
        Use:   "demo",
        Short: "Charm libraries tech demo",
        Run:   runDemo,
    }
    parent.AddCommand(cmd)
}
```

## First Implementation: Charm Tech Demo

### Goal
Showcase **all Charm library capabilities** in an interactive demo.

### Demo Menu Structure

```
cly demo
├── [1] Bubbletea Basics     # Model-Update-View pattern
├── [2] Bubbles Showcase     # All pre-built components
├── [3] Huh Forms            # Form/prompt examples
├── [4] Lipgloss Gallery     # Styling showcase
└── [5] Full Demo            # Combined experience
```

### Feature Demonstrations

#### 1. Bubbletea Basics
- [ ] Simple list navigation (cursor movement)
- [ ] Key bindings (vim-style: j/k, arrows)
- [ ] State management demo
- [ ] Commands and messages

#### 2. Bubbles Showcase
| Component | Demo |
|-----------|------|
| Spinner | Loading states with different spinner types |
| TextInput | Single-line input with validation |
| TextArea | Multi-line text editor |
| Table | Sortable data table |
| Progress | Animated progress bar |
| Paginator | Dot-style and numeric pagination |
| Viewport | Scrollable content pager |
| List | Filterable list with fuzzy search |
| FilePicker | File/directory browser |
| Timer/Stopwatch | Time tracking demo |
| Help | Auto-generated keybinding help |

#### 3. Huh Forms
- [ ] Input field with validation
- [ ] Text area (multi-line)
- [ ] Select (single choice)
- [ ] MultiSelect (checkboxes)
- [ ] Confirm (yes/no)
- [ ] Multi-page form wizard
- [ ] Dynamic forms (options change based on input)
- [ ] Theme showcase (Charm, Dracula, Catppuccin, Base16)
- [ ] Spinner integration

#### 4. Lipgloss Gallery
- [ ] Color profiles (ANSI 16, 256, TrueColor)
- [ ] Adaptive colors (light/dark detection)
- [ ] Text formatting (bold, italic, underline, etc.)
- [ ] Borders (normal, rounded, thick, double, custom)
- [ ] Padding and margins
- [ ] Text alignment
- [ ] Width/height constraints
- [ ] JoinHorizontal/JoinVertical layouts
- [ ] Table rendering
- [ ] List rendering (nested, custom enumerators)
- [ ] Tree rendering

#### 5. Full Demo
Interactive experience combining all above:
1. Form to collect user preferences (Huh)
2. Display styled summary (Lipgloss)
3. Show loading spinner (Bubbles)
4. Present results in styled table (Lipgloss table)
5. Allow navigation through results (Bubbletea)

## Config Structure (Viper/YAML)

```yaml
# ~/.config/cly/config.yaml
app:
  name: cly
  debug: false

theme:
  style: charm  # charm | dracula | catppuccin | base16 | default

demo:
  default_section: full
```

### Config Features to Demonstrate
- [x] YAML config file (`~/.config/cly/config.yaml`)
- [x] Environment variable override (`CLY_APP_DEBUG=true`)
- [x] Flag binding (`--debug`)
- [x] Config file watching (live reload)
- [x] Default values

## CLI Structure

```
cly [flags]
├── demo [flags]           # Run the tech demo
│   ├── --section <name>   # Jump to specific section
│   └── --theme <name>     # Override theme
├── config                 # Config management
│   ├── init               # Create default config
│   ├── show               # Display current config
│   └── edit               # Open config in $EDITOR
├── version                # Show version info
└── help                   # Auto-generated help
```

## Non-Functional Requirements

1. **No TOML** - Config must be YAML (Viper)
2. **Modular** - Adding commands requires no core changes
3. **Testable** - Interfaces allow mocking
4. **Cross-platform** - macOS, Linux, Windows
5. **Single binary** - No external dependencies

## Future Modules (Post-Demo)

The modular structure allows adding utilities:
- `cly uuid` - Generate UUIDs
- `cly json` - JSON pretty-print/validate
- `cly encode` - Base64/URL encoding
- `cly http` - Quick HTTP client with TUI
- etc.

## Development Setup

```bash
# Init
go mod init github.com/user/cly
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/huh
go get github.com/charmbracelet/lipgloss

# Run
go run main.go demo

# Build
go build -o cly
```

You have access to gitmcp of all those libs
