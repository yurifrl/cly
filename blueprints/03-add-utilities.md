# Phase 3: Add More Utilities

**Goal**: Build utilities you'll actually use

**Deliverable**: 3-5 working utility commands

**Time**: Ongoing (add as needed)

---

## Strategy

**Copy the UUID pattern** for each new utility:
- Copy `modules/uuid/` directory
- Adapt for new functionality
- Register in `cmd/root.go`

Choose utilities based on what you need daily.

---

## Utility Ideas

### JSON Formatter
**Command**: `cly json`

**Uses**: Viewport (for scrolling large output)

**What it does**:
- Pretty-print JSON from stdin or file
- Validate JSON
- Extract fields with jq-like syntax

**Copy from**: `modules/uuid/`

---

### Base64 Encoder/Decoder
**Command**: `cly encode`

**Uses**: Form (for input collection)

**What it does**:
- Encode/decode base64
- URL encoding
- Hex encoding

**Copy from**: `modules/uuid/`

---

### HTTP Requester
**Command**: `cly http`

**Uses**: Table (for response display), Form (for request building)

**What it does**:
- Quick HTTP GET/POST
- Display response in table
- Save/load request templates

**Copy from**: `modules/uuid/`

---

### File Picker
**Command**: `cly pick`

**Uses**: Filetree component

**What it does**:
- Interactive file browser
- Select file and copy path to clipboard
- Filter by extension

**Copy from**: `modules/uuid/`

---

### Password Generator
**Command**: `cly pass`

**Uses**: Textinput (for options), Progress (for strength meter)

**What it does**:
- Generate secure passwords
- Custom length/character sets
- Show strength meter

**Copy from**: `modules/uuid/`

---

## Pattern: Adding a Utility

### Step: Copy Module Structure

```bash
cp -r modules/uuid modules/json
```

### Step: Edit cmd.go

```go
package json

import (
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "json",
		Short: "Format and validate JSON",
		RunE:  run,
	})
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	return err
}
```

### Step: Implement Logic

Edit `json.go` with your implementation:
- Keep model/init/update/view pattern
- Use appropriate Charm components
- Focus on utility, not demo

### Step: Register

```go
// cmd/root.go
import "cly/modules/json"

func init() {
	uuid.Register(RootCmd)
	json.Register(RootCmd)  // Add
}
```

### Step: Test

```bash
go run main.go json
```

---

## When to Use Each Charm Component

**List** - Multiple choice selection
- File picker
- Option menus
- Command selection

**Table** - Structured data display
- HTTP response headers
- File listings
- Key-value pairs

**Textinput** - Single value input
- Search queries
- File paths
- Simple configuration

**Form (Huh)** - Multiple inputs
- Request builders
- Configuration wizards
- Multi-field forms

**Viewport** - Scrollable content
- Large JSON output
- Log viewing
- Documentation

**Progress** - Long operations
- File downloads
- Batch operations
- Strength meters

**Spinner** - Indefinite wait
- API calls
- File operations
- Background tasks

---

## Example: JSON Utility

**Full implementation** of `modules/json/`:

```go
// modules/json/cmd.go
package json

import (
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "json",
		Short: "Format and validate JSON",
		Long:  "Pretty-print JSON from stdin or file",
		RunE:  run,
	}

	cmd.Flags().StringP("file", "f", "", "Input file")
	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")
	p := tea.NewProgram(initialModel(file))
	_, err := p.Run()
	return err
}
```

```go
// modules/json/json.go
package json

import (
	"encoding/json"
	"io/ioutil"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	viewport viewport.Model
	content  string
	ready    bool
}

func initialModel(file string) model {
	var data []byte
	var err error

	if file != "" {
		data, err = ioutil.ReadFile(file)
	} else {
		data, err = ioutil.ReadAll(os.Stdin)
	}

	if err != nil {
		return model{content: "Error reading input: " + err.Error()}
	}

	// Pretty-print JSON
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return model{content: "Invalid JSON: " + err.Error()}
	}

	pretty, _ := json.MarshalIndent(obj, "", "  ")

	return model{content: string(pretty)}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.SetContent(m.content)
			m.ready = true
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Press q to quit")

	return style.Render(m.viewport.View()) + "\n" + help
}
```

**Test**:
```bash
echo '{"name":"test"}' | go run main.go json
go run main.go json -f data.json
```

---

## Success Criteria

✅ Each utility does real work
✅ Uses Charm components appropriately
✅ Can pipe input/output (Unix philosophy)
✅ You actually use these commands

---

## Next Phase

Continue to `04-readme.md` to document your utilities.
