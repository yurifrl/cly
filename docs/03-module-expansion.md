# Phase 3: Module Expansion

**Goal**: Add 3 more commands using the spinner pattern

**Deliverable**: 4 working commands (`spinner`, `textinput`, `list`, `table`)

**Time**: 1 hour (20 minutes per module)

---

## Strategy

**Copy the spinner pattern** for each new module:
1. Copy `modules/spinner/` directory
2. Rename package
3. Adapt code from reference example
4. Register in `cmd/root.go`

---

## Module 1: TextInput

### Reference
`references/bubbletea/examples/textinput/main.go`

### Steps

```bash
cp -r modules/spinner modules/textinput
```

**Edit `modules/textinput/cmd.go`**:
- Change package to `textinput`
- Change command name to `"textinput"`
- Update Short/Long descriptions

**Edit `modules/textinput/textinput.go`**:
- Change package to `textinput`
- Replace spinner code with textinput code from reference
- Keep structure: `model`, `initialModel()`, `Init()`, `Update()`, `View()`

**Register in `cmd/root.go`**:
```go
import (
	// ...
	"cly/modules/textinput"
)

func init() {
	spinner.Register(RootCmd)
	textinput.Register(RootCmd)  // Add this
}
```

**Test**:
```bash
go run main.go textinput
```

---

## Module 2: List

### Reference
`references/bubbletea/examples/list-simple/main.go`

### Steps

```bash
cp -r modules/spinner modules/list
```

**Edit `modules/list/cmd.go`**:
- Change package to `list`
- Change command name to `"list"`
- Update descriptions

**Edit `modules/list/list.go`**:
- Change package to `list`
- Adapt list-simple code from reference
- Keep MVC structure

**Register in `cmd/root.go`**:
```go
import (
	// ...
	"cly/modules/list"
)

func init() {
	spinner.Register(RootCmd)
	textinput.Register(RootCmd)
	list.Register(RootCmd)  // Add this
}
```

**Test**:
```bash
go run main.go list
```

---

## Module 3: Table

### Reference
`references/bubbletea/examples/table/main.go`

### Steps

```bash
cp -r modules/spinner modules/table
```

**Edit `modules/table/cmd.go`**:
- Change package to `table`
- Change command name to `"table"`
- Update descriptions

**Edit `modules/table/table.go`**:
- Change package to `table`
- Adapt table code from reference
- Keep MVC structure

**Register in `cmd/root.go`**:
```go
import (
	// ...
	"cly/modules/table"
)

func init() {
	spinner.Register(RootCmd)
	textinput.Register(RootCmd)
	list.Register(RootCmd)
	table.Register(RootCmd)  // Add this
}
```

**Test**:
```bash
go run main.go table
```

---

## Final cmd/root.go

```go
package cmd

import (
	"github.com/spf13/cobra"
	"cly/pkg/style"
	"cly/modules/spinner"
	"cly/modules/textinput"
	"cly/modules/list"
	"cly/modules/table"
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
	textinput.Register(RootCmd)
	list.Register(RootCmd)
	table.Register(RootCmd)
}

func Execute() error {
	return RootCmd.Execute()
}
```

---

## Directory Structure After Phase 3

```
cly/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   └── root.go
├── modules/
│   ├── spinner/
│   │   ├── cmd.go
│   │   └── spinner.go
│   ├── textinput/
│   │   ├── cmd.go
│   │   └── textinput.go
│   ├── list/
│   │   ├── cmd.go
│   │   └── list.go
│   └── table/
│       ├── cmd.go
│       └── table.go
└── pkg/
    └── style/
        └── theme.go
```

---

## Test All Commands

```bash
cly --help        # Shows all 4 commands
cly spinner       # Works
cly textinput     # Works
cly list          # Works
cly table         # Works
```

---

## Success Criteria

✅ All 4 commands appear in `--help`
✅ Each command runs interactively
✅ All commands can be quit with 'q'
✅ No import errors

---

## Next Phase

Continue to `04-polish.md` for production polish (refactoring to MVC, version, README).
