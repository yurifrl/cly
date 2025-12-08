# Module Template

**Use this template to add new commands quickly.**

---

## Quick Steps

1. Copy existing module: `cp -r modules/spinner modules/<newname>`
2. Edit files (replace `spinner` with `<newname>`)
3. Register in `cmd/root.go`
4. Test

---

## Template Files

### File: `modules/<newname>/cmd.go`

```go
package <newname>

import (
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "<newname>",
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

---

### File: `modules/<newname>/model.go` (Phase 4 MVC)

```go
package <newname>

type model struct {
	// State fields
	quitting bool
}

func initialModel() model {
	return model{}
}
```

---

### File: `modules/<newname>/update.go` (Phase 4 MVC)

```go
package <newname>

import tea "github.com/charmbracelet/bubbletea"

func (m model) Init() tea.Cmd {
	return nil
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
```

---

### File: `modules/<newname>/view.go` (Phase 4 MVC)

```go
package <newname>

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	return "Hello from <newname>\nPress q to quit\n"
}
```

---

## Registration in `cmd/root.go`

**Add import**:
```go
import (
	// ...
	"cly/modules/<newname>"
)
```

**Add to init()**:
```go
func init() {
	// ...
	<newname>.Register(RootCmd)
}
```

---

## Reference Examples Map

| Command | Reference |
|---------|-----------|
| `spinner` | `references/bubbletea/examples/spinner` |
| `textinput` | `references/bubbletea/examples/textinput` |
| `textarea` | `references/bubbletea/examples/textarea` |
| `list` | `references/bubbletea/examples/list-simple` |
| `table` | `references/bubbletea/examples/table` |
| `progress` | `references/bubbletea/examples/progress-static` |
| `form` | `references/bubbletea/examples/credit-card-form` |
| `viewport` | `references/bubbletea/examples/pager` |
| `stopwatch` | `references/bubbletea/examples/stopwatch` |
| `timer` | `references/bubbletea/examples/timer` |

---

## Naming Conventions

### Command Names
- **Lowercase** only
- **No spaces** or hyphens for simple commands
- **Hyphens** for multi-word: `long-running`, `split-view`

### Package Names
- Same as command name (lowercase, no hyphens)
- Use underscores if needed: `long_running`

### File Names
- `cmd.go` - Always this name
- `<component>.go` - Phase 2-3 (one file)
- `model.go`, `update.go`, `view.go` - Phase 4 (MVC split)

---

## Example: Adding Progress Bar

### 1. Copy Structure
```bash
cp -r modules/spinner modules/progress
```

### 2. Edit `modules/progress/cmd.go`
```go
package progress

import (
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "progress",
		Short: "Progress bar demo",
		Long:  "Animated progress bar from Bubbles library",
		RunE:  run,
	})
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	return err
}
```

### 3. Adapt from Reference
**Open**: `references/bubbletea/examples/progress-static/main.go`

**Copy** model/init/update/view to `modules/progress/progress.go`

### 4. Register
```go
// cmd/root.go
import "cly/modules/progress"

func init() {
	// ...
	progress.Register(RootCmd)
}
```

### 5. Test
```bash
go run main.go progress
```

---

## Checklist

- [ ] Command name is lowercase
- [ ] Package name matches command
- [ ] `Register()` function exists
- [ ] Added to `cmd/root.go` imports
- [ ] Added to `cmd/root.go` init()
- [ ] Tested: command appears in `--help`
- [ ] Tested: command runs interactively
- [ ] Can quit with 'q'

---

## Tips

1. **Start simple** - One file in Phase 2-3, refactor to MVC in Phase 4
2. **Copy working code** - Adapt from reference examples
3. **Test early** - Run after each change
4. **Keep it focused** - One demo per command
5. **Follow patterns** - Look at existing modules

---

## Advanced: Adding Flags

```go
var flagValue string

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "example",
		Short: "Example with flags",
		RunE:  run,
	}

	// Add flag
	cmd.Flags().StringVarP(&flagValue, "type", "t", "default", "Type of example")

	parent.AddCommand(cmd)
}
```

Use `flagValue` in your model/view logic.
