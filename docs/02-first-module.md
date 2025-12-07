# Phase 2: First Module (Spinner)

**Goal**: Add first working command

**Deliverable**: `cly spinner` runs interactive demo

**Time**: 30 minutes

---

## Reference Material

Adapt from: `references/bubbletea/examples/spinner/main.go`

**Strategy**: Copy working code, keep it in ONE file for now. Refactor later in Phase 4.

---

## Files to Create

### 1. Module Command Registration

```bash
mkdir -p modules/spinner
```

**File: `modules/spinner/cmd.go`** (~20 lines)

```go
package spinner

import (
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "spinner",
		Short: "Animated spinner showcase",
		Long:  "Demonstrates various spinner animations from the Bubbles library",
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

---

### 2. Bubbletea Implementation

**File: `modules/spinner/spinner.go`** (~80 lines)

**Open**: `references/bubbletea/examples/spinner/main.go`

**Copy** the following into `modules/spinner/spinner.go`:
- Type `model struct`
- Function `initialModel()`
- Method `Init()`
- Method `Update()`
- Method `View()`

**Adapt**:
1. Change package from `main` to `spinner`
2. Remove `main()` function (we have `run()` in `cmd.go`)
3. Keep all Bubbletea logic intact

**Template**:
```go
package spinner

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
)

type model struct {
	spinner spinner.Model
	quitting bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return model{spinner: s}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
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

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}
	return fmt.Sprintf("\n\n   %s Loading...\n\n   Press q to quit\n\n", m.spinner.View())
}
```

---

### 3. Register Module with Root

**Edit: `cmd/root.go`**

Add import:
```go
import (
	"github.com/spf13/cobra"
	"cly/pkg/style"
	"cly/modules/spinner"  // Add this
)
```

Add init function:
```go
func init() {
	spinner.Register(RootCmd)
}
```

**Full file** should look like:
```go
package cmd

import (
	"github.com/spf13/cobra"
	"cly/pkg/style"
	"cly/modules/spinner"
)

var RootCmd = &cobra.Command{
	Use:   "cly",
	Short: style.TitleStyle.Render("Charm Libraries Showcase"),
	Long: `Interactive demos of Bubbletea, Bubbles, Huh, and Lipgloss.

Each command demonstrates a different Charm component:
  • spinner   - Animated loading spinners
  • textinput - Text input fields
  • list      - Selectable lists
  • table     - Data tables

Press 'q' or Ctrl+C to quit any demo.`,
}

func init() {
	spinner.Register(RootCmd)
}

func Execute() error {
	return RootCmd.Execute()
}
```

---

### 4. Install Bubbles

```bash
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/bubbletea
```

---

## Test

```bash
# Check help
go run main.go --help

# Should now show spinner command
go run main.go spinner
```

**Expected**:
- Animated spinner appears
- Press 'q' to quit
- Clean exit

---

## Directory Structure After Phase 2

```
cly/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   └── root.go
├── modules/
│   └── spinner/
│       ├── cmd.go
│       └── spinner.go
└── pkg/
    └── style/
        └── theme.go
```

---

## Troubleshooting

**Import error**: Run `go mod tidy`

**Spinner not animating**: Terminal may not support animations (try different terminal)

**Can't quit**: Make sure `tea.KeyMsg` handling is correct

---

## Success Criteria

✅ `cly spinner` runs
✅ Spinner animates
✅ Press 'q' quits cleanly
✅ `cly --help` lists spinner command

---

## Next Phase

Continue to `03-module-expansion.md` to add more commands using the same pattern.
