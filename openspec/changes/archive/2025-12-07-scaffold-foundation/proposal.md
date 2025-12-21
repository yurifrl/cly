# Change: Scaffold Foundation Infrastructure

## Why
CLY needs the minimal foundation infrastructure to function as a working CLI application. This establishes the project structure, dependency management, and basic command framework that all future modules will build upon. Without this foundation, no commands can be implemented.

## What Changes
- Initialize Go module with proper naming (`github.com/yurifrl/cly`)
- Install core dependencies (Cobra for CLI, Lipgloss for styling)
- Create minimal entry point (`main.go`)
- Implement root command with styled help output (`cmd/root.go`)
- Add shared styling utilities (`pkg/style/theme.go`)
- Establish project directory structure (cmd/, pkg/, modules/ placeholder)

## Impact
- **Affected specs**: Creates new capability `cli-foundation`
- **Affected code**:
  - New: `main.go`, `go.mod`, `cmd/root.go`, `pkg/style/theme.go`
  - Establishes: Project structure for modular command registration
- **Dependencies**: Adds `github.com/spf13/cobra` and `github.com/charmbracelet/lipgloss`
- **Testing**: Deliverable is `go run main.go --help` showing styled output
