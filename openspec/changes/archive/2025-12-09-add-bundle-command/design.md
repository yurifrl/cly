# Design: add-bundle-command

## Architecture

### Component Overview

```
pkg/store/              ← shared infrastructure
├── store.go            # Store interface + DuckDB impl
└── store_test.go

modules/bundle/         ← bundle module
├── bundle.go           # Register, subcommands
├── bundler.go          # Bundler interface
├── brew.go             # BrewBundler (no Store)
├── go.go               # GoBundler
├── js.go               # JsBundler
├── python.go           # PythonBundler
└── common.go           # file parsing
```

### Store Interface

```go
// pkg/store/store.go
type Store interface {
    List(namespace string) ([]string, error)
    Add(namespace, key string) error
    Remove(namespace, key string) error
}
```

Generic namespace/key store. Bundle uses namespace = bundler type ("go", "js", "python").

### Bundler Interface

```go
// modules/bundle/bundler.go
type Bundler interface {
    Name() string           // "go", "js", "python"
    DefaultFile() string    // ~/.config/Gofile
    CheckDeps() error       // verify tool exists
    Install(pkg string) error
    Uninstall(pkg string) error
}
```

### Dependency Injection

Store created in `cmd/root.go`, injected into bundle.Register():

```go
// cmd/root.go
func init() {
    db := store.New("~/.config/cly/cly.db")
    bundle.Register(RootCmd, db)
}
```

### Database Schema

```sql
CREATE TABLE packages (
    type    VARCHAR NOT NULL,
    name    VARCHAR NOT NULL,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (type, name)
);
```

Location: `~/.config/cly/cly.db`

### Subcommand Flows

**`cly bundle [type]`** — full sync:
1. Parse bundle file → desired
2. Query Store → installed
3. Uninstall (installed - desired), remove from Store
4. Install desired, add to Store

**`cly bundle check [type]`** — dry-run:
1. Parse bundle file → desired
2. Query Store → installed
3. Print diff
4. Exit 0 if in sync, 1 if changes needed

**`cly bundle cleanup [type]`** — remove only:
1. Parse bundle file → desired
2. Query Store → installed
3. Uninstall (installed - desired), remove from Store

### Brew Special Case

Brew delegates entirely to `brew bundle`:
- `cly bundle brew` → `brew bundle --file=~/.config/Brewfile`
- `cly bundle check brew` → `brew bundle check --file=~/.config/Brewfile`
- `cly bundle cleanup brew` → `brew bundle cleanup --file=~/.config/Brewfile --force`

No Store interaction needed — brew tracks its own state.

### State Sync Behavior

**Out of sync (DB says installed, but isn't):** Try uninstall, fails or no-op, log warning, delete from DB anyway. Self-heals.

**Not in DB but installed:** Treated as manual install, untouched.

## Trade-offs

**DuckDB vs text files:** DuckDB provides transactional consistency and query capability. Overkill for simple lists, but Store is shared infrastructure usable by other modules.

**Subcommands vs flags:** `check` and `cleanup` as subcommands mirrors `brew bundle` CLI. Cleaner than `--dry-run` and `--no-install` flags.

**Injection vs global:** Store injection keeps modules testable and decoupled. Could use global singleton but injection is cleaner.
