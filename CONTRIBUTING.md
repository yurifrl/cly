# Contributing to CLY

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

**Requirements:** Go 1.22+, [Task](https://taskfile.dev) (optional but recommended)

```bash
git clone git@github.com:yurifrl/cly.git
cd cly
go mod download
task build    # or: go build -o dist/cly .
task test     # or: go test ./...
```

## Project Structure

```
cmd/root.go              # Root command, module registration
main.go                  # Entry point
modules/                 # Self-contained command modules
pkg/                     # Shared utilities (config, store, style, etc.)
```

Each module is self-contained under `modules/`. Modules don't import each other — shared code lives in `pkg/`.

## Adding a Module

1. Create `modules/mymodule/cmd.go`:

```go
package mymodule

import "github.com/spf13/cobra"

func Register(parent *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "mymodule",
        Short: "Description",
        RunE:  run,
    }
    parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
    return nil
}
```

2. Register in `cmd/root.go`:

```go
mymodule.Register(RootCmd)
```

**Key conventions:**
- Always use `RunE` (not `Run`) so errors propagate
- TUI commands use Bubbletea (`tea.NewProgram`)
- Use `pkg/style` for terminal styling

## Testing

Integration tests preferred over mocks:

```bash
go test ./...                          # All tests
go test ./modules/mcp/... -v           # Specific module
go test ./pkg/config/... -run TestLoad # Specific test
```

## Submitting Changes

1. Fork the repo and create a branch
2. Make your changes with tests
3. Run `task test` to verify
4. Open a PR with a clear description

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep modules self-contained
- Error messages should be actionable

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
