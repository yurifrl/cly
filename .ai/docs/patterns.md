---
triggers: [pattern, how to, example, bubbletea, cobra, store]
---

# Quick Patterns

## Bubbletea MVC

```go
type model struct { }
func (m model) Init() tea.Cmd { }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { }
func (m model) View() string { }
```

Update returns NEW model (immutable). View is pure function.

**Examples:** modules/demo/spinner/, modules/uuid/

## Module Registration

```go
// modules/mymod/cmd.go
func Register(parent *cobra.Command) {
    parent.AddCommand(&cobra.Command{
        Use: "mymod",
        RunE: run,
    })
}

// cmd/root.go
func init() {
    mymod.Register(RootCmd)
}
```

**Why:** Zero coupling between modules.

## Config Access

```go
cfg := config.Get()
value := config.GetString("modules.mymod.setting")
```

Secrets auto-resolved from `op://` refs.

## Store Usage

```go
store := store.New("~/.local/share/cly/cly.db")
store.Add("namespace", "key")
store.List("namespace")
store.Remove("namespace", "key")
```

Namespace = bundler type (go, js, python).

## Error Wrapping

```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

Always wrap with context.
