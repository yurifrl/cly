# Phase 2: First Utility (UUID)

**Goal**: Add first working utility command

**Deliverable**: `cly uuid` generates UUIDs

**Time**: 30 minutes

---

## Why UUID?

Real utility that:
- Does useful work (generate UUIDs)
- Uses Charm components (textinput/list for type selection)
- Simple enough to establish pattern
- You'll actually use it

---

## Files to Create

### Module Command Registration

```bash
mkdir -p modules/uuid
```

**File: `modules/uuid/cmd.go`**

```go
package uuid

import (
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "uuid",
		Short: "Generate UUIDs",
		Long:  "Interactive UUID generator with multiple format options",
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

### Bubbletea Implementation

**File: `modules/uuid/uuid.go`**

```go
package uuid

import (
	"fmt"
	"github.com/google/uuid"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

type model struct {
	list     list.Model
	choice   string
	quitting bool
	generated string
}

func initialModel() model {
	items := []list.Item{
		item("UUID v4 (random)"),
		item("UUID v7 (time-ordered)"),
		item("Multiple (5x)"),
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 10)
	l.Title = "Generate UUID"
	l.SetShowHelp(false)

	return model{list: l}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
				// Generate UUID based on choice
				switch m.choice {
				case "UUID v4 (random)":
					m.generated = uuid.New().String()
				case "UUID v7 (time-ordered)":
					m.generated = uuid.Must(uuid.NewV7()).String()
				case "Multiple (5x)":
					for i := 0; i < 5; i++ {
						m.generated += uuid.New().String() + "\n"
					}
				}
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting && m.generated == "" {
		return "Cancelled.\n"
	}
	if m.generated != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		return style.Render(m.generated) + "\n"
	}
	return "\n" + m.list.View()
}
```

---

### Register Module with Root

**Edit: `cmd/root.go`**

Add import:
```go
import (
	"github.com/spf13/cobra"
	"cly/pkg/style"
	"cly/modules/uuid"  // Add this
)
```

Add init function:
```go
func init() {
	uuid.Register(RootCmd)
}
```

---

### Install Dependencies

```bash
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/bubbletea
go get github.com/google/uuid
```

---

## Test

```bash
# Check help
go run main.go --help

# Run UUID generator
go run main.go uuid
```

**Expected**:
- List appears with 3 options
- Select one with arrow keys + enter
- UUID printed to stdout
- Press 'q' to cancel

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
│   └── uuid/
│       ├── cmd.go
│       └── uuid.go
└── pkg/
    └── style/
        └── theme.go
```

---

## Troubleshooting

**Import error**: Run `go mod tidy`

**List not showing**: Check terminal size

**UUID not copying**: Pipe to clipboard: `cly uuid | pbcopy` (macOS)

---

## Success Criteria

✅ `cly uuid` runs
✅ List is interactive
✅ UUID generated and printed
✅ `cly --help` lists uuid command

---

## Next Phase

Continue to `03-add-utilities.md` to add more useful commands.
