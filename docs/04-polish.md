# Phase 4: Polish

**Goal**: Production-ready quality

**Deliverable**: Can share publicly

**Time**: 1.5 hours

---

## Task 1: Refactor Spinner to MVC (30 min)

**Current**: `modules/spinner/spinner.go` (one file)

**Target**: Split into model/view/update

### Create Files

**File: `modules/spinner/model.go`**
```go
package spinner

import (
	"github.com/charmbracelet/bubbles/spinner"
)

type model struct {
	spinner  spinner.Model
	quitting bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return model{spinner: s}
}
```

**File: `modules/spinner/update.go`**
```go
package spinner

import tea "github.com/charmbracelet/bubbletea"

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
```

**File: `modules/spinner/view.go`**
```go
package spinner

import "fmt"

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}
	return fmt.Sprintf("\n\n   %s Loading...\n\n   Press q to quit\n\n", m.spinner.View())
}
```

**Delete**: `modules/spinner/spinner.go`

**Test**: `cly spinner` still works

---

## Task 2: Add Version Command (15 min)

**File: `cmd/version.go`**
```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"cly/pkg/style"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(style.TitleStyle.Render("cly v0.1.0"))
		fmt.Println(style.SubtleStyle.Render("Charm Libraries Showcase"))
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
```

**Test**: `cly version`

---

## Task 3: Add List Command (15 min)

**File: `cmd/list.go`**
```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"cly/pkg/style"
)

var listCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all available commands",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(style.TitleStyle.Render("Available Commands\n"))

		fmt.Println("  • spinner   - Animated loading spinners")
		fmt.Println("  • textinput - Text input fields")
		fmt.Println("  • list      - Selectable lists")
		fmt.Println("  • table     - Data tables")
		fmt.Println()
		fmt.Println(style.SubtleStyle.Render("Run 'cly <command>' to try a demo"))
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
```

**Test**: `cly ls`

---

## Task 4: Create README.md (30 min)

**File: `README.md`**
```markdown
# CLY - Charm Libraries Showcase

Interactive CLI demonstrating Bubbletea, Bubbles, Huh, and Lipgloss.

## Installation

```bash
go install github.com/yourusername/cly@latest
```

Or build from source:
```bash
git clone https://github.com/yourusername/cly
cd cly
go build
```

## Usage

```bash
# Show all commands
cly --help
cly ls

# Try demos
cly spinner       # Animated spinners
cly textinput     # Text input
cly list          # Selectable list
cly table         # Data table
```

## Commands

| Command | Description |
|---------|-------------|
| `spinner` | Animated loading spinners |
| `textinput` | Text input field demo |
| `list` | Selectable list navigation |
| `table` | Data table display |
| `version` | Show version |
| `ls` | List all commands |

## Architecture

Modular design following NSX-CLI patterns:
- Each command is a self-contained module
- Zero coupling between modules
- Easy to add new commands

```
modules/
├── spinner/
│   ├── cmd.go       # Registration
│   ├── model.go     # State
│   ├── update.go    # Logic
│   └── view.go      # Display
└── <others>/
```

## Adding a Module

See `docs/module-template.md` for template.

1. Copy existing module
2. Adapt Bubbletea code
3. Register in `cmd/root.go`

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling

## License

MIT
```

---

## Task 5: Build Binary (10 min)

```bash
# Build for current platform
go build -o cly

# Test binary
./cly --help
./cly spinner

# Build for multiple platforms (optional)
GOOS=darwin GOARCH=arm64 go build -o cly-darwin-arm64
GOOS=linux GOARCH=amd64 go build -o cly-linux-amd64
GOOS=windows GOARCH=amd64 go build -o cly-windows-amd64.exe
```

---

## Final Directory Structure

```
cly/
├── main.go
├── go.mod
├── go.sum
├── README.md
├── cmd/
│   ├── root.go
│   ├── version.go
│   └── list.go
├── modules/
│   ├── spinner/
│   │   ├── cmd.go
│   │   ├── model.go
│   │   ├── update.go
│   │   └── view.go
│   ├── textinput/
│   │   ├── cmd.go
│   │   └── textinput.go
│   ├── list/
│   │   ├── cmd.go
│   │   └── list.go
│   └── table/
│       ├── cmd.go
│       └── table.go
├── pkg/
│   └── style/
│       └── theme.go
└── docs/
    ├── 00-overview.md
    ├── 01-foundation.md
    ├── 02-first-module.md
    ├── 03-module-expansion.md
    ├── 04-polish.md
    └── module-template.md
```

---

## Success Criteria

✅ Spinner refactored to MVC pattern
✅ `cly version` works
✅ `cly ls` lists commands
✅ README.md exists with examples
✅ Binary builds successfully
✅ All commands work in binary

---

## Optional: Config Support

See SPEC.md for Viper configuration implementation. Can defer to later release.

---

## Release Checklist

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build`
- [ ] README complete
- [ ] Version tagged: `git tag v0.1.0`
- [ ] Binary uploaded to GitHub Releases
- [ ] Demo GIFs/screenshots added

---

## Next Steps

- Add more modules (progress, form, viewport, textarea)
- Implement config system (Viper)
- Add tests
- CI/CD setup
- Publish to Homebrew/pkg managers
