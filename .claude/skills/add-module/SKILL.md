---
name: add-module
description: Create new demo or utility modules following CLY project patterns. Use when adding TUI demonstration modules to modules/demo/ or utility modules to modules/, following Bubbletea/Bubbles conventions with proper Cobra CLI integration.
---

# Add Module Skill

Automates creation of new modules in the CLY project following established patterns.

## When to Use This Skill

- User wants to add a new demo module showcasing a Bubbletea component
- User wants to create a new utility command
- User mentions "create a module", "add a command", "new demo"

## Module Types

### Demo Modules (`modules/demo/<name>/`)
**Purpose**: Showcase Charm UI components and patterns
**Examples**: chat, spinner, table, list-simple (48 total)
**Parent**: Registered under `demo` namespace

### Utility Modules (`modules/<name>/`)
**Purpose**: Provide real functionality
**Examples**: uuid (UUID generator)
**Parent**: Registered directly under root command

## Step-by-Step Workflow

### 1. Determine Module Type
Ask user if unclear:
- "Is this a demo (showcase component) or utility (real functionality)?"

### 2. Find Reference (for demos)
- Check if component exists in `references/bubbletea/examples/<name>/`
- Read the reference implementation
- Note initialization code in main() function

### 3. Create Directory Structure
```bash
# For demo:
mkdir -p modules/demo/<name>

# For utility:
mkdir -p modules/<name>
```

### 4. Create cmd.go (Command Registration)

**Template for demos:**
```go
package <packagename>

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "<name>",
		Short: "<description>",
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

**Add tea.Program options if needed:**
- `tea.WithAltScreen()` - For fullscreen demos
- `tea.WithMouseAllMotion()` - For mouse tracking
- `tea.WithReportFocus()` - For focus/blur events

### 5. Create Implementation File

**Extract from reference:**
1. Copy type definitions (model struct, custom types)
2. Copy Init(), Update(), View() methods
3. Create initialModel() from main() function's initialization code
4. Remove unused imports (fmt, os, log often unused after main() removal)

**Template:**
```go
package <packagename>

import (
	tea "github.com/charmbracelet/bubbletea"
	// Component imports as needed
)

type model struct {
	// State fields
}

func initialModel() model {
	// Initialization from reference's main()
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

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

func (m model) View() string {
	return "Your UI\n"
}
```

### 6. Register Module

**For demos** - Edit `modules/demo/cmd.go`:
```go
import (
	yourmodule "github.com/yurifrl/cly/modules/demo/your-module"
)

func init() {
	// ... existing registrations
	yourmodule.Register(DemoCmd)
}
```

**For utilities** - Edit `cmd/root.go`:
```go
import (
	"github.com/yurifrl/cly/modules/yourutil"
)

func init() {
	// ... existing registrations
	yourutil.Register(RootCmd)
}
```

### 7. Validation Checklist

- [ ] Compiles: `go build`
- [ ] Shows in help: `go run main.go --help` or `go run main.go demo --help`
- [ ] Runs: `go run main.go <command>` (or `go run main.go demo <name>`)
- [ ] Quits cleanly with 'q' or Ctrl+C
- [ ] No unused imports

## Common Patterns

### Package Naming
- Directory with hyphens: `list-simple/`
- Package name with underscores: `package list_simple`
- Import alias: `listsimple "github.com/yurifrl/cly/modules/demo/list-simple"`

### Extracting initialModel()

**In reference main():**
```go
func main() {
	s := spinner.New()
	s.Spinner = spinner.Dot
	m := model{spinner: s}
	tea.NewProgram(m).Run()
}
```

**Extract to:**
```go
func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return model{spinner: s}
}
```

### Helper Functions
If reference has helpers (like getPackages(), filter(), etc.), copy them to the implementation file.

## Examples to Reference

**Simple**: `modules/demo/spinner/` - Basic component
**Complex**: `modules/demo/list-fancy/` - Multiple files (delegate.go, randomitems.go)
**Utility**: `modules/uuid/` - Real functionality with list UI
**Advanced**: `modules/demo/chat/` - Multiple components (textarea + viewport)

## Quick Reference Commands

```bash
# Test compilation
go build

# View help
go run main.go --help
go run main.go demo --help

# Run demo
go run main.go demo <name>

# Run utility
go run main.go <name>

# Clean dependencies
go mod tidy
```

## Best Practices

1. **Start from reference** - All 48 Bubbletea examples available in references/
2. **Copy existing module** - Fastest way to get structure right
3. **Test incrementally** - Build and run after each file
4. **Clean imports early** - Remove fmt/os/log before testing
5. **Follow naming** - Hyphens in names, underscores in packages

## Troubleshooting

### Build Errors
- **"undefined: initialModel"** → Function not created or private (make sure it's `initialModel`, not `InitialModel`)
- **"unused import"** → Remove it from imports
- **"package name mismatch"** → Check hyphens vs underscores

### Runtime Errors
- **"could not open TTY"** → Normal in non-interactive shells, try in terminal
- **Component not responding** → Check Update() delegates to component's Update()
- **Can't quit** → Verify KeyMsg handling for "q" and "ctrl+c"

---

For detailed templates and examples, see `docs/module-template.md`.
