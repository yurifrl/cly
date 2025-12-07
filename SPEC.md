# CLY - Modular Charm CLI Specification

**Purpose**: Modular Go CLI showcasing Charm libraries (Bubbletea, Bubbles, Huh, Lipgloss) with production-ready architecture patterns from NSX-CLI.

---

## Technology Stack

| Library | Purpose |
|---------|---------|
| **Cobra** | CLI framework (commands, flags, help) |
| **Viper** | Configuration management (YAML, env vars) |
| **Bubbletea** | TUI framework (Elm Architecture) |
| **Bubbles** | Pre-built TUI components |
| **Huh** | Forms and prompts |
| **Lipgloss** | Styling (colors, borders, layout) |

---

## Architecture: Everything is a Command

```bash
cly spinner          # Each Charm component is a direct command
cly table
cly list
cly textinput
cly form
cly progress
cly viewport
cly textarea

cly config init      # Utility commands
cly config show
cly version
```

**No "demo" parent command** - This prepares for future utility commands beyond demos.

---

## Project Structure

```
cly/
├── main.go                  # Entry point
├── go.mod
├── cmd/
│   └── root.go              # Root command, registers all modules
├── modules/                 # Each module = one Charm component
│   ├── spinner/
│   │   ├── cmd.go           # Exports Register(parent)
│   │   ├── model.go         # Bubbletea model
│   │   ├── view.go          # Bubbletea view
│   │   └── update.go        # Bubbletea update
│   ├── table/
│   │   ├── cmd.go
│   │   └── ...
│   ├── list/
│   ├── textinput/
│   ├── form/
│   ├── progress/
│   ├── viewport/
│   ├── textarea/
│   └── config/
│       └── cmd.go           # Config management commands
├── pkg/                     # Shared utilities
│   ├── config/
│   │   └── config.go        # Viper-based config loader
│   ├── style/
│   │   └── theme.go         # Lipgloss styles
│   └── ui/
│       ├── components.go    # Reusable components
│       ├── help.go          # Help display
│       └── status.go        # Status messages
├── config/
│   └── config.yaml          # Default configuration
└── references/              # Working examples (for development)
    ├── bubbletea/examples/  # Official Bubbletea examples
    ├── nsx-cli/             # NSX-CLI modular architecture reference
    └── soft-serve/          # Advanced Charm usage
```

---

## Key Modularity Patterns (from NSX-CLI)

### 1. Command Registration (Query Command Pattern)

**Single registration point**:
```go
// cmd/root.go
package cmd

import (
    "github.com/spf13/cobra"
    "cly/modules/spinner"
    "cly/modules/table"
    "cly/modules/list"
)

var RootCmd = &cobra.Command{
    Use:   "cly",
    Short: "Charm libraries showcase CLI",
    PersistentPreRunE: initConfig,
}

func Execute() {
    // Modules register themselves - add line per module
    spinner.Register(RootCmd)
    table.Register(RootCmd)
    list.Register(RootCmd)

    RootCmd.Execute()
}
```

### 2. Module Self-Containment (Locality & Behavior)

**Each module is independent**:
```go
// modules/spinner/cmd.go
package spinner

import "github.com/spf13/cobra"

func Register(parent *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "spinner",
        Short: "Spinner component showcase",
        Long:  `Demonstrates Bubbles spinner component with multiple types`,
        RunE:  runSpinner,
    }

    cmd.Flags().StringVarP(&spinnerType, "type", "t", "dot", "Spinner type")
    parent.AddCommand(cmd)
}

func runSpinner(cmd *cobra.Command, args []string) error {
    p := tea.NewProgram(initialModel())
    _, err := p.Run()
    return err
}
```

### 3. Shared Utilities (Package-Oriented Design)

**Zero coupling between modules**:
```go
// pkg/style/theme.go
package style

import "github.com/charmbracelet/lipgloss"

var (
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("212"))

    SuccessStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("42"))
)

func ApplyTheme(themeName string) {
    // charm, dracula, catppuccin
}
```

**Usage in any module**:
```go
// modules/spinner/view.go
import "cly/pkg/style"

func (m model) View() string {
    return style.TitleStyle.Render("Spinner Demo") + "\n" + m.spinner.View()
}
```

### 4. Configuration (Viper + YAML)

```yaml
# config/config.yaml
app:
  name: cly
  debug: false

theme:
  style: charm  # charm | dracula | catppuccin

modules:
  spinner:
    default_type: dot
  table:
    default_height: 10
```

```go
// pkg/config/config.go
package config

import "github.com/spf13/viper"

type Config struct {
    App struct {
        Name  string
        Debug bool
    }
    Theme struct {
        Style string
    }
    Modules map[string]map[string]interface{}
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.AddConfigPath("$HOME/.config/cly")
    viper.AddConfigPath(".")

    // Env var support: CLY_APP_DEBUG=true
    viper.SetEnvPrefix("CLY")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

---

## Module Template

**Every module follows MVC pattern (Bubbletea)**:

```go
// modules/<name>/cmd.go
package <name>

import (
    "github.com/spf13/cobra"
    tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "<name>",
        Short: "<description>",
        Long:  `<detailed help>`,
        RunE:  run<Name>,
    }

    cmd.Flags().StringVarP(&flag, "flag", "f", "default", "description")
    parent.AddCommand(cmd)
}

func run<Name>(cmd *cobra.Command, args []string) error {
    p := tea.NewProgram(initialModel())
    _, err := p.Run()
    return err
}

// modules/<name>/model.go
package <name>

import tea "github.com/charmbracelet/bubbletea"

type model struct {
    // State fields
}

func initialModel() model {
    return model{}
}

func (m model) Init() tea.Cmd {
    return nil
}

// modules/<name>/update.go
package <name>

import tea "github.com/charmbracelet/bubbletea"

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    }
    return m, nil
}

// modules/<name>/view.go
package <name>

import "cly/pkg/style"

func (m model) View() string {
    return style.TitleStyle.Render("Title") + "\n" + "content"
}
```

---

## Reference Material Mapping

Each module adapts working examples from `references/bubbletea/examples/`:

| Command | Reference Example |
|---------|-------------------|
| `cly spinner` | `references/bubbletea/examples/spinner` |
| `cly textinput` | `references/bubbletea/examples/textinput` |
| `cly list` | `references/bubbletea/examples/list-simple` |
| `cly table` | `references/bubbletea/examples/table` |
| `cly progress` | `references/bubbletea/examples/progress-static` |
| `cly form` | `references/bubbletea/examples/credit-card-form` |
| `cly viewport` | `references/bubbletea/examples/pager` |
| `cly textarea` | `references/bubbletea/examples/textarea` |

**Don't reinvent** - adapt and integrate existing working code.

---

## Implementation Phases

### Phase 1: Foundation
**Files**:
- `main.go` - Entry point
- `go.mod` - Dependencies
- `cmd/root.go` - Root command with global flags
- `pkg/config/config.go` - Config loader
- `config/config.yaml` - Default config
- `pkg/style/theme.go` - Shared styles

**Test**: `go run main.go --help`

### Phase 2: First Module (Spinner)
**Files**:
- `modules/spinner/cmd.go`
- `modules/spinner/model.go`
- `modules/spinner/view.go`
- `modules/spinner/update.go`

**Reference**: `references/bubbletea/examples/spinner/main.go`

**Test**: `cly spinner` runs working demo

### Phase 3: Additional Modules
**Priority order**:
1. spinner ✓ (Phase 2)
2. textinput
3. list
4. table
5. progress
6. form
7. viewport
8. textarea

**Process**:
- Copy `modules/spinner/` structure
- Adapt reference example
- Register in `cmd/root.go`

### Phase 4: Shared Components
**Files**:
- `pkg/ui/components.go` - Generic wrappers
- `pkg/ui/help.go` - Help display
- `pkg/ui/status.go` - Status messages

**Refactor**: Use shared components in modules

### Phase 5: Config Commands
**Files**:
- `modules/config/cmd.go`
  - `cly config init`
  - `cly config show`
  - `cly config set <key> <value>`

### Phase 6: Polish
**Tasks**:
- Add `cly version`
- Add `cly list` (list all commands)
- Create README.md
- Add help text examples
- Cross-platform testing

---

## Dependencies

```bash
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/huh
```

---

## Success Criteria

✅ **Modular** - Adding command = zero changes to existing modules
✅ **Self-contained** - Each module has all code in its directory
✅ **Reusable** - Shared utilities prevent duplication
✅ **Scalable** - Grows from 5 to 50+ commands easily
✅ **Working demos** - Each command shows Charm capabilities
✅ **Config-driven** - Behavior customizable via YAML
✅ **Single binary** - No external runtime dependencies

---

## Design Principles

1. **Single registration point** (`cmd/root.go`)
2. **Module autonomy** (no inter-module dependencies)
3. **Shared consistency** (common utilities for UX)
4. **Type safety** (interfaces, generics)
5. **Encapsulation** (independently testable modules)
6. **Configuration locality** (module-specific config sections)

This architecture enables parallel development without conflicts.
