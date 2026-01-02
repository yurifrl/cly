# Git Worktree Module

TUI + CLI worktree manager for cly.

## Architecture

**Page-switching model** - no overlays, full page replacement.

### Why not overlays?
- ANSI cursor tricks are buggy, delete lines
- True overlays need fullscreen (`tea.WithAltScreen()`)
- Page switching is clean, reliable, inline

### Model structure

```go
type page int

const (
    pageMenu page = iota
    pageAdd
    pageSwitch
    pagePalette
)

type model struct {
    page     page
    prevPage page  // for returning from palette

    // Per-page state
    paletteInput    textinput.Model
    paletteFiltered []command
    paletteCursor   int
    menuCursor      int
}
```

### Update pattern

Global keys checked first, then dispatch to page handler:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Global: ctrl+p from any page
        if msg.String() == "ctrl+p" && m.page != pagePalette {
            m.prevPage = m.page
            m.page = pagePalette
            m.paletteInput = newPaletteInput() // reset
            return m, textinput.Blink
        }

        // Dispatch to page
        switch m.page {
        case pagePalette:
            return m.updatePalette(msg)
        case pageMenu:
            return m.updateMenu(msg)
        }
    }
    return m, nil
}
```

### View pattern

```go
func (m model) View() string {
    switch m.page {
    case pagePalette:
        return m.viewPalette()
    case pageMenu:
        return m.viewMenu()
    }
    return ""
}
```

### Palette behavior

- All keystrokes go to `textinput` (filtering)
- Only special keys intercepted: `↑/↓`, `enter`, `esc`
- `esc` returns to `prevPage`
- `enter` navigates to selected command's target page

```go
func (m model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc":
        m.page = m.prevPage
        return m, nil
    case "enter":
        selected := m.paletteFiltered[m.paletteCursor]
        m.page = selected.page
        return m, nil
    case "down":
        m.paletteCursor++
        return m, nil
    }
    // Default: update text input
    m.paletteInput, cmd = m.paletteInput.Update(msg)
    m.filterPalette()
    return m, cmd
}
```

### Components used

- `bubbles/textinput` - palette filter input
- `lipgloss` - styling, borders
- Manual cursor rendering (no `bubbles/list`)

Reference: `examples/palette/main.go`

## Commands

```
cly worktree              # TUI main menu
cly worktree add          # TUI with branch autocomplete
cly worktree add -b <branch>
cly worktree switch       # TUI list (default: cd)
cly worktree switch -p    # print path
cly worktree switch -z    # zellij tab
cly worktree remove
cly worktree prune
cly worktree push / push-all
cly worktree delete [--remote]
cly worktree status [--all]
cly worktree list
```

## TUI

### Main Menu
```
┌────────────────────────────────┐
│  Worktree                      │
│  ──────────────────────────    │
│  > Add worktree                │
│    Switch worktree             │
│    Remove worktree             │
│    Status                      │
│    Push                        │
│    Push all                    │
│    Prune                       │
│    Delete                      │
│                                │
│  ↑/↓ • enter • / • ctrl+p      │
└────────────────────────────────┘
```

### Add (autocomplete)
```
┌────────────────────────────────┐
│  Worktree > Add                │
│  ──────────────────────────    │
│  Branch: yurifrl/feat/a█       │
│          ┌──────────────────┐  │
│          │ yurifrl/feat/auth│  │
│          │ yurifrl/feat/api │  │
│          └──────────────────┘  │
│                                │
│  [ ] Create new branch (c)     │
│                                │
│  tab • ↑/↓ • esc back          │
└────────────────────────────────┘
```

### Switch (search)
```
┌────────────────────────────────┐
│  Worktree > Switch             │
│  ──────────────────────────    │
│  > main (current directory)    │
│    feat-auth                   │
│    fix-login                   │
│                                │
│  enter cd • z zellij • p print │
│  ↑/↓ • / • esc back            │
└────────────────────────────────┘
```

### Status
```
┌────────────────────────────────┐
│  Worktree > Status             │
│  ──────────────────────────    │
│  > main        ✓ clean         │
│    feat-auth   ● 3 modified    │
│    fix-login   ✓ clean         │
│                                │
│  ↑/↓ • / • esc back            │
└────────────────────────────────┘
```

## Keys

**Global** (every page):
| Key | Action |
|-----|--------|
| `ctrl+p` | Open palette |
| `?` | Help |
| `g` | Go home (menu) |
| `ctrl+c` | Quit |

**Navigation** (lists/menus):
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `enter` | Select |
| `esc` | Back |
| `q` | Quit |

**Search** (lists only):
| Key | Action |
|-----|--------|
| `/` | Enter search mode |
| `n/N` | Next/prev match |
| `esc` | Exit search |

Note: `/` activates search to avoid key conflicts.

**Palette**:
- All typing filters commands
- `↑/↓` or `ctrl+n/k` navigate
- `enter` select, `esc` close

## Header

Every screen has:
```
│  Worktree > Switch             │
│  ──────────────────────────    │
```

Command palette (`ctrl+p`):
```
┌────────────────────────────────┐
│  > █                           │
│                                │
│  > add      Add worktree       │
│    switch   Switch worktree    │
│    remove   Remove worktree    │
│    status   Worktree status    │
│                                │
│  ↑/↓ • enter • esc             │
└────────────────────────────────┘
```

## Config

```yaml
worktree:
  directory: ".worktrees"
  branch_prefix: "yurifrl/feat/"
```

## Structure

```
modules/worktree/
├── cmd.go           # Cobra registration
├── worktree.go      # Git operations
├── worktree_test.go
├── tui.go           # Main model + global keys + vim nav
├── tui_palette.go   # Command palette (ctrl+p)
├── tui_help.go      # Help screen (?)
├── tui_menu.go      # Main menu
├── tui_add.go       # Branch autocomplete
├── tui_switch.go    # List with search
├── tui_select.go    # Generic selector
└── tui_status.go
```

`tui.go` wraps all screens, handles:
- Global keys (`ctrl+p`, `?`, `g`, `q`)
- Vim nav (`h/j/k/l`)
- Breadcrumb header
- Screen stack (back with `h`)

## Core API

```go
type Worktree struct {
    Name, Path, Branch, Commit string
}

func List() ([]Worktree, error)
func Add(branch string, create bool) error
func Remove(name string, force bool) error
func Prune() error
func GetBranches() ([]string, error)
func GetStatus(name string) (string, error)
func Push(name string) error
func Delete(name string, remote bool) error
func SwitchTo(name, mode string) error
func EnsureGitignore() (bool, error)
func BranchToFolder(branch string) string
```

## Gitignore

First run checks `.worktrees` in `.gitignore`, prompts if missing.

## References

- `modules/demo/autocomplete/`
- `modules/demo/list-simple/`
- `.opencode/skill/git-worktrees/SKILL.md`
