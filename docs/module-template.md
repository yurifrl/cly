# Module Template

**Use this template to add new commands quickly.**

---

## Module Categories

### Demo Modules (UI Component Showcases)
**Location**: `modules/demo/<name>/`
**Purpose**: Demonstrate Charm components and patterns
**Examples**: `chat`, `spinner`, `table`, `list-simple`

**When to use**: Showcasing UI components, TUI patterns, Bubbletea features

### Utility Modules (Real Functionality)
**Location**: `modules/<name>/`
**Purpose**: Provide actual utility commands
**Examples**: `uuid` (UUID generator)

**When to use**: Commands users will actually use for work

---

## Quick Steps

### For Demo Modules
1. Find reference: `references/bubbletea/examples/<component>/`
2. Copy pattern: `cp -r modules/demo/spinner modules/demo/<newname>`
3. Adapt implementation from reference
4. Register in `modules/demo/cmd.go` init()
5. Test

### For Utility Modules
1. Copy pattern: `cp -r modules/uuid modules/<newname>`
2. Implement functionality
3. Register in `cmd/root.go` init()
4. Test

---

## Demo Module Template

### File: `modules/demo/<name>/cmd.go`

```go
package <packagename>  // Use underscores for hyphens: list_simple for list-simple

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "<name>",
		Short: "<short description>",
		Long:  "<detailed description>",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
```

### File: `modules/demo/<name>/<name>.go`

```go
package <packagename>

import (
	tea "github.com/charmbracelet/bubbletea"
	// Add component imports as needed:
	// "github.com/charmbracelet/bubbles/spinner"
	// "github.com/charmbracelet/bubbles/list"
	// "github.com/charmbracelet/bubbles/table"
	// "github.com/charmbracelet/lipgloss"
)

type model struct {
	// Component state
	quitting bool
	err      error
}

func initialModel() model {
	// Initialize your model here
	// Extract this from reference example's main() function
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
	// Or return component's Init: spinner.Tick, textarea.Blink, etc.
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	return "Your UI here\nPress q to quit\n"
}
```

---

## Registration Patterns

### Demo Module Registration

**File**: `modules/demo/cmd.go`

```go
import (
	// ...
	yourmodule "github.com/yurifrl/cly/modules/demo/your-module"
)

func init() {
	// ...
	yourmodule.Register(DemoCmd)
}
```

### Utility Module Registration

**File**: `cmd/root.go`

```go
import (
	// ...
	"github.com/yurifrl/cly/modules/yourutil"
)

func init() {
	uuid.Register(RootCmd)
	demo.Register(RootCmd)
	yourutil.Register(RootCmd)  // Add here
}
```

---

## Bubbletea Program Options

Some demos require special options when creating the tea.Program:

### AltScreen (Fullscreen Mode)
```go
func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```
**Use when**: Demo should use alternate screen buffer (fullscreen, eyes, cellbuffer)
**Examples**: `modules/demo/fullscreen/`, `modules/demo/eyes/`

### Mouse Support
```go
p := tea.NewProgram(initialModel(), tea.WithMouseAllMotion())
```
**Use when**: Demo needs mouse tracking
**Example**: `modules/demo/mouse/`

### Focus Reporting
```go
p := tea.NewProgram(initialModel(), tea.WithReportFocus())
```
**Use when**: Demo needs to know when terminal gains/loses focus
**Example**: `modules/demo/focus-blur/`

### Input Filtering
```go
p := tea.NewProgram(initialModel(), tea.WithFilter(filterFunc))
```
**Use when**: Need to intercept/modify messages before Update()
**Example**: `modules/demo/prevent-quit/`

---

## Reference Examples (48 Available)

All 48 Bubbletea examples are in `references/bubbletea/examples/` and `modules/demo/`:

### Core Components
| Demo | Shows | Reference |
|------|-------|-----------|
| `spinner` | Animated loading | `references/bubbletea/examples/spinner` |
| `list-simple` | Selection lists | `references/bubbletea/examples/list-simple` |
| `table` | Data tables | `references/bubbletea/examples/table` |
| `textinput` | Single-line input | `references/bubbletea/examples/textinput` |
| `textarea` | Multi-line input | `references/bubbletea/examples/textarea` |
| `progress-static` | Progress bars | `references/bubbletea/examples/progress-static` |

### Advanced
| Demo | Shows | Reference |
|------|-------|-----------|
| `chat` | Textarea + Viewport | `references/bubbletea/examples/chat` |
| `file-picker` | File selection | `references/bubbletea/examples/file-picker` |
| `credit-card-form` | Complex forms | `references/bubbletea/examples/credit-card-form` |
| `split-editors` | Multiple panes | `references/bubbletea/examples/split-editors` |

**All 48 examples** are available - explore `modules/demo/` for implementations.

---

## Adapting Reference Examples

### Step-by-Step Process

1. **Find reference**: `references/bubbletea/examples/<component>/main.go`
2. **Read main() function**: This has initialization code
3. **Extract to initialModel()**: Move setup from main() to initialModel()
4. **Copy Model implementation**: Copy type definitions, Init(), Update(), View()
5. **Clean imports**: Remove `fmt`, `os`, `log` if unused
6. **Create cmd.go**: Use template above with Register() function

### Example: Adapting Spinner

**Reference**: `references/bubbletea/examples/spinner/main.go`

**Extract this from main()**:
```go
s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
return model{spinner: s}
```

**Becomes initialModel()**:
```go
func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{spinner: s}
}
```

---

## Naming Conventions

### Command Names
- **Lowercase** only
- **Hyphens** for multi-word: `list-simple`, `credit-card-form`, `altscreen-toggle`

### Package Names
- **Lowercase**, **no hyphens**
- Use underscores: `list_simple`, `credit_card_form`, `altscreen_toggle`
- Go converts hyphens automatically during import

### File Names
- `cmd.go` - Always this name (command registration)
- `<name>.go` - Main implementation (e.g., `spinner.go`, `list-simple.go`)
- Additional files: `delegate.go`, `helpers.go`, `types.go` (if needed)

---

## Checklist

### Before Implementation
- [ ] Decided: demo or utility module?
- [ ] Found reference example (if demo)
- [ ] Command name chosen (lowercase, hyphens if multi-word)

### During Implementation
- [ ] Created directory in correct location
- [ ] Created cmd.go with Register() function
- [ ] Created implementation file with initialModel()
- [ ] Package name matches conventions
- [ ] Imports are clean (no unused)

### After Implementation
- [ ] Registered in parent cmd.go init()
- [ ] Import added to parent cmd.go
- [ ] Compiles: `go build`
- [ ] Appears in help: `go run main.go --help` or `go run main.go demo --help`
- [ ] Runs: `go run main.go <command>`
- [ ] Quits cleanly with 'q' or Ctrl+C

---

## Tips

1. **Start with existing demos** - 48 working examples to learn from
2. **Copy working code** - Don't reinvent, adapt from references
3. **Test frequently** - Build and run after each change
4. **Keep it simple** - Single file until complexity demands splitting
5. **Use shared styles** - Import `pkg/style` for consistent theming
6. **Follow the pattern** - Look at 3-4 similar modules before starting

---

## Troubleshooting

### "undefined: initialModel"
- Make sure initialModel() function exists in implementation file
- Check it's exported (lowercase 'i' makes it package-private)

### "package name mismatch"
- Directory name with hyphens → package name with underscores
- Example: `list-simple/` → `package list_simple`

### "unused import"
- Remove `fmt`, `os`, `log` if not actually used
- Check your View() and Update() functions
- Common after removing main() function

### "command not showing in help"
- Verify Register() called in parent's init()
- Check import path is correct
- Run `go mod tidy`

---

## Advanced Patterns

### Multiple Files (Complex Modules)
```
modules/demo/list-fancy/
├── cmd.go           # Command registration
├── list-fancy.go    # Model and main logic
├── delegate.go      # Custom item delegate
└── randomitems.go   # Helper functions
```

### With Flags
```go
var demoType string

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Demo with flag",
		RunE:  run,
	}

	cmd.Flags().StringVarP(&demoType, "type", "t", "default", "Demo type")
	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	// Use demoType variable in initialModel()
	p := tea.NewProgram(initialModel(demoType))
	_, err := p.Run()
	return err
}
```

### With Required Args
```go
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "download <url>",
		Short: "Download with progress",
		Args:  cobra.ExactArgs(1),  // Require 1 argument
		RunE:  run,
	}
	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	url := args[0]
	p := tea.NewProgram(initialModel(url))
	_, err := p.Run()
	return err
}
```

---

## Current Project Stats

- **Total demo modules**: 48
- **Total utility modules**: 1 (uuid)
- **Reference examples**: 48 in `references/bubbletea/examples/`
- **All demos working**: ✅ Yes
