# Phase 1: Foundation

**Goal**: Minimal working CLI with `--help` support

**Deliverable**: `cly --help` shows styled output

**Time**: 15 minutes

---

## Files to Create

### 1. Initialize Go Module

```bash
go mod init github.com/yourusername/cly
```

### 2. Install Dependencies

```bash
go get github.com/spf13/cobra
go get github.com/charmbracelet/lipgloss
```

---

## File: `main.go` (10 lines)

```go
package main

import (
	"cly/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

---

## File: `pkg/style/theme.go` (15 lines)

```bash
mkdir -p pkg/style
```

```go
package style

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	SubtleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)
```

---

## File: `cmd/root.go` (30 lines)

```bash
mkdir -p cmd
```

```go
package cmd

import (
	"github.com/spf13/cobra"
	"cly/pkg/style"
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

func Execute() error {
	return RootCmd.Execute()
}
```

---

## Test

```bash
go run main.go --help
```

**Expected output**: Styled help text with title and description

**Troubleshooting**:
- Import error? Run `go mod tidy`
- Style not showing? Check terminal supports colors

---

## Directory Structure After Phase 1

```
cly/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   └── root.go
└── pkg/
    └── style/
        └── theme.go
```

---

## Success Criteria

✅ `go run main.go --help` works
✅ Output is styled (bold title)
✅ No errors
✅ Build succeeds: `go build`

---

## Next Phase

Continue to `02-first-module.md` to add the spinner command.
