# Design: add-bundle-command

## Architecture

### Module Structure

```
modules/bundle/
├── cmd.go          # Register(), root command, type dispatch
├── bundler.go      # Bundler interface, common state/parsing
├── brew.go         # BrewBundler implementation
├── go.go           # GoBundler implementation
├── js.go           # JsBundler implementation
└── python.go       # PythonBundler implementation
```

### Bundler Interface

```go
type Bundler interface {
    Name() string
    DefaultFile() string
    StateFile() string
    CheckDeps() error
    Install(pkg string) error
    Uninstall(pkg string) error
}
```

Each bundler implements this interface. `brew` is special-cased since `brew bundle` already handles the full sync.

### Execution Flow

```
cly bundle [type] [flags]
    │
    ├─► resolve type (default: brew)
    ├─► load bundler for type
    ├─► check dependencies (tool exists?)
    ├─► optionally open editor (--edit)
    ├─► parse bundle file
    ├─► load state file
    ├─► diff: to_install, to_remove
    ├─► if --dry-run: print diff, exit
    ├─► execute removes
    ├─► execute installs
    └─► save state file
```

### Bundle Files

| Type   | File                    | State File                   |
|--------|-------------------------|------------------------------|
| brew   | ~/.config/Brewfile      | (managed by brew)            |
| go     | ~/.config/Gofile        | ~/.config/go_bundle_state    |
| js     | ~/.config/Jsfile        | ~/.config/js_bundle_state    |
| python | ~/.config/Pythonfile    | ~/.config/python_bundle_state|

### Type-Specific Behavior

**brew**: Calls `brew bundle` directly. No custom state tracking.

**go**:
- Detects mise for GOPATH/GOBIN setup
- Falls back to `go env GOPATH` if no mise
- Uses `go install pkg@latest`

**js**:
- Uses `bun install -g` / `bun remove -g`
- Normalizes GitHub shorthand: `user/repo` → `github:user/repo`
- Preserves scoped packages: `@scope/pkg` unchanged

**python**:
- Uses `uv tool install` / `uv tool uninstall`
- Supports extras syntax: `pkg[extra1,extra2]`

### Error Handling

- Missing dependency tool: error with install instructions
- Missing bundle file: error (no implicit creation)
- Install failure: continue with remaining, report failures at end
- State saved only for successful installs

## Trade-offs

**Shell-out vs native**: Shelling out to brew/bun/go/uv is simpler and maintains compatibility with existing bundle file formats. Native implementation would require reimplementing package resolution logic.

**Single state file per type**: Matches existing scripts. Alternative (single combined state) rejected for simplicity.

**No TUI**: Bundle operations are batch jobs. Progress output via plain text with colors matches existing behavior.
