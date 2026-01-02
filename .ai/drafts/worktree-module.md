# Git Worktree Module

TUI + CLI worktree manager for cly.

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
│  j/k • l select • / • ctrl+p   │
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
│  tab • n/p • h back            │
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
│  l cd • z zellij • o print     │
│  j/k • / • h back              │
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
│  j/k • / • h back              │
└────────────────────────────────┘
```

## Global Keys (every screen)

| Key | Action |
|-----|--------|
| `ctrl+p` | Command palette |
| `?` | Help |
| `g` | Go home |
| `q` | Quit |

## Vim Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Down |
| `k` / `↑` | Up |
| `l` / `enter` | Select / enter |
| `h` / `esc` | Back |

## List Keys

| Key | Action |
|-----|--------|
| `/` | Search mode |
| `n` | Next match |
| `p` | Prev match |

## Input Keys (autocomplete)

| Key | Action |
|-----|--------|
| `tab` | Accept suggestion |
| `n` | Next suggestion |
| `p` | Prev suggestion |

## Navigation

Header on every screen:
```
│  Worktree > Switch             │
│  ──────────────────────────    │
```

Command palette (`ctrl+p`):
```
┌────────────────────────────────┐
│  > █                           │
│  add      Add worktree         │
│  switch   Switch worktree      │
│  remove   Remove worktree      │
│  status   Worktree status      │
│                                │
│  l select • h cancel           │
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
