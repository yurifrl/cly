# Blueprint: Bundle Command

Unified declarative package management for multiple ecosystems.

## Overview

`cly bundle [type]` syncs packages from declarative files in `~/.config/`. Wraps existing bundler logic into a single interface.

## Usage

```bash
cly bundle [type]           # install + cleanup (default: brew)
cly bundle check [type]     # show what would change
cly bundle cleanup [type]   # remove unlisted packages only
```

Types: `brew`, `go`, `js`, `python`, `all`

## Flags

```
--file, -f      Override default bundle file path
--verbose, -v   Show detailed output
```

Mirrors `brew bundle` simplicity.

## Bundle Files

Location: `~/.config/`

| Type   | File        | Format                          |
|--------|-------------|---------------------------------|
| brew   | Brewfile    | `brew "pkg"`, `cask "app"`      |
| go     | Gofile      | `github.com/user/repo/cmd/bin`  |
| js     | Jsfile      | `@scope/pkg` or `user/repo`     |
| python | Pythonfile  | `package` or `package[extras]`  |

## State & Cleanup

Cleanup is the core value prop: bundle file is the source of truth, anything not in it gets removed.

### State Tracking

| Type   | How it knows what's installed                     |
|--------|---------------------------------------------------|
| brew   | `brew bundle` queries brew directly               |
| go     | DuckDB: `~/.config/cly/cly.db`                    |
| js     | DuckDB: `~/.config/cly/cly.db`                    |
| python | DuckDB: `~/.config/cly/cly.db`                    |

**brew**: No state needed. `brew bundle cleanup` compares Brewfile against `brew list` output directly.

**go/js/python**: DuckDB tracks installed packages. Store lives in `pkg/store`—shared infrastructure, injected into bundle module.

```sql
CREATE TABLE packages (
    type    VARCHAR NOT NULL,  -- 'go', 'js', 'python'
    name    VARCHAR NOT NULL,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (type, name)
);
```

After successful install, insert row. On cleanup, diff table against bundle file—anything in DB but not in bundle gets uninstalled then deleted from table.

**Location**: `~/.config/cly/cly.db` (shared app database)

**Out of sync**: DB says package X installed but it's not (user manually removed). On cleanup we try uninstall, fails or no-op, log warning, delete from DB anyway. Self-heals. Reverse: package installed but not in DB—untouched (manual install).

### Subcommands

**`cly bundle [type]`** (default): install + cleanup
```
1. Read bundle file → desired
2. Query DB → installed
3. to_remove = installed - desired
4. to_install = desired (reinstall is idempotent)
5. Uninstall each in to_remove, delete from DB
6. Install each in to_install, insert to DB
```

**`cly bundle check [type]`**: dry-run, show diff
```
1. Read bundle file → desired
2. Query DB → installed
3. Print: would install X, would remove Y
4. Exit 0 if in sync, exit 1 if changes needed
```

**`cly bundle cleanup [type]`**: remove only, no install
```
1. Read bundle file → desired
2. Query DB → installed
3. to_remove = installed - desired
4. Uninstall each, delete from DB
```

### Per-Type Cleanup

**brew**
```bash
brew bundle cleanup --file=~/.config/Brewfile --force
```
Handles formulas, casks, taps. `--force` actually removes (without it just lists).

**go**
```bash
# derive binary name from package path
# github.com/foo/bar/cmd/baz → baz
# github.com/foo/bar → bar
rm $GOBIN/{binary_name}
```
No uninstall command exists for go—just delete the binary.

**js**
```bash
bun remove -g {package}
```
Works for npm packages and github: prefixed packages.

**python**
```bash
uv tool uninstall {package}
```
Removes the tool and its isolated environment.

### Edge Cases

- Failed uninstall: log warning, delete from DB anyway, continue
- Package in DB but binary missing: delete from DB, no uninstall attempt
- First run (empty DB): install only, no cleanup

## Implementation

### File Structure

```
pkg/store/
├── store.go        # Store interface + DuckDB implementation
└── store_test.go

modules/bundle/
├── bundle.go       # Register, root command, check, cleanup subcommands
├── bundler.go      # Bundler interface
├── brew.go         # brew bundle wrapper (no Store needed)
├── go.go           # GoBundler
├── js.go           # JsBundler
├── python.go       # PythonBundler
└── common.go       # file parsing, colors
```

### Core Logic

Sync flow (non-brew):
```go
func Sync(bundler Bundler, store Store, bundleFile string) error {
    desired := parseFile(bundleFile)
    installed := store.List(bundler.Name())

    toRemove := diff(installed, desired)
    for _, pkg := range toRemove {
        bundler.Uninstall(pkg)
        store.Remove(bundler.Name(), pkg)
    }

    for _, pkg := range desired {
        bundler.Install(pkg)  // idempotent
        store.Add(bundler.Name(), pkg)
    }
}
```

### Command Registration

```go
// modules/bundle/bundle.go
func Register(root *cobra.Command, store store.Store) {
    cmd := &cobra.Command{
        Use:   "bundle [type]",
        Short: "Sync packages from declarative files",
    }
    cmd.Flags().StringP("file", "f", "", "bundle file path")
    cmd.Flags().BoolP("verbose", "v", false, "verbose output")

    cmd.AddCommand(checkCmd(store))
    cmd.AddCommand(cleanupCmd(store))
    // default action is sync
    cmd.RunE = syncCmd(store)

    root.AddCommand(cmd)
}

// cmd/root.go
func init() {
    db := store.New("~/.config/cly/cly.db")
    bundle.Register(RootCmd, db)
}
```

### Interfaces

```go
// pkg/store/store.go
type Store interface {
    List(namespace string) ([]string, error)
    Add(namespace, key string) error
    Remove(namespace, key string) error
}

// modules/bundle/bundler.go
type Bundler interface {
    Name() string                 // "go", "js", "python"
    DefaultFile() string          // ~/.config/Gofile, etc.
    CheckDeps() error             // verify tool exists
    Install(pkg string) error
    Uninstall(pkg string) error
}
```

**Injection**: Store created in `cmd/root.go`, passed to `bundle.Register(root, store)`.

Brew is special—delegates entirely to `brew bundle`, no Store needed.

## Behavior Notes

- `brew`: calls `brew bundle` with `--cleanup` flag when cleanup enabled
- `go`: handles mise integration for GOPATH/GOBIN, removes binaries on cleanup
- `js`: normalizes GitHub shorthand (`user/repo` → `github:user/repo`)
- `python`: uses `uv tool install/uninstall`
- `all`: runs each bundler in sequence, continues on individual failures

## Dependencies

**Go packages**:
- `github.com/marcboeker/go-duckdb` — DuckDB driver (for pkg/store)

**External tools** (checked at runtime via `CheckDeps()`):
- `brew` — brew type
- `bun` — js type
- `go` — go type (+ mise for GOBIN detection)
- `uv` — python type

## Test Plan

- Unit tests for file parsing (comments, blank lines, whitespace)
- Unit tests for Store (Add, Remove, List)
- Unit tests for diff logic: DB vs bundle file
- Test `check` returns correct exit code
- Test `cleanup` removes from DB after uninstall
- Integration test per bundler type
