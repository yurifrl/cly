# Blueprint: Bundle Command

Unified declarative package management for multiple ecosystems.

## Overview

`cly bundle [type]` syncs packages from declarative files in `~/.config/`. Wraps existing bundler logic into a single interface.

## Usage

```bash
cly bundle           # defaults to brew
cly bundle brew      # Brewfile → brew bundle
cly bundle go        # Gofile → go install
cly bundle js        # Jsfile → bun install -g
cly bundle python    # Pythonfile → uv tool install
cly bundle all       # run all bundlers
```

## Flags

```
--edit, -e      Open bundle file in $EDITOR before sync (default: true)
--no-edit       Skip editor, just sync
--dry-run       Show what would change without doing it
--file, -f      Override default file path
```

## Bundle Files

Location: `~/.config/`

| Type   | File        | Format                          |
|--------|-------------|---------------------------------|
| brew   | Brewfile    | `brew "pkg"`, `cask "app"`      |
| go     | Gofile      | `github.com/user/repo/cmd/bin`  |
| js     | Jsfile      | `@scope/pkg` or `user/repo`     |
| python | Pythonfile  | `package` or `package[extras]`  |

State files track installed packages: `~/.config/{type}_bundle_state`

## Implementation

### File Structure

```
modules/bundle/
├── bundle.go       # Register, root bundle command
├── brew.go         # brew bundle wrapper
├── go.go           # go install logic
├── js.go           # bun install logic
├── python.go       # uv tool logic
└── common.go       # shared: colors, state, file parsing
```

### Core Logic

Each bundler:
- Reads package list from file (skip comments/blanks)
- Compares against state file
- Installs new packages
- Removes packages not in file
- Updates state file

### Command Registration

```go
// bundle.go
func Register(root *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "bundle [type]",
        Short: "Sync packages from declarative files",
        Args:  cobra.MaximumNArgs(1),
        RunE:  runBundle,
    }
    cmd.Flags().BoolP("edit", "e", true, "open file in editor first")
    cmd.Flags().Bool("no-edit", false, "skip editor")
    cmd.Flags().Bool("dry-run", false, "show changes only")
    cmd.Flags().StringP("file", "f", "", "override bundle file path")

    root.AddCommand(cmd)
}

func runBundle(cmd *cobra.Command, args []string) error {
    bundleType := "brew"
    if len(args) > 0 {
        bundleType = args[0]
    }
    // dispatch to appropriate bundler
}
```

### Bundler Interface

```go
type Bundler interface {
    Name() string
    DefaultFile() string
    StateFile() string
    Install(pkg string) error
    Uninstall(pkg string) error
    CheckDeps() error  // verify tool exists (brew, bun, go, uv)
}
```

## Behavior Notes

- `brew` type calls `brew bundle` directly (already declarative)
- `go` type handles mise integration for GOPATH/GOBIN
- `js` type normalizes GitHub shorthand (`user/repo` → `github:user/repo`)
- `python` type uses `uv tool install/uninstall`

## Dependencies

- brew (for brew type)
- bun (for js type)
- go + mise (for go type)
- uv (for python type)

## Test Plan

- Unit tests for file parsing
- Unit tests for state diff logic
- Integration test per bundler type
