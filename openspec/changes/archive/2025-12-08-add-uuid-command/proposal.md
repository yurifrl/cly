# Change: Add UUID Generator Command

## Why
CLY needs its first working utility command to establish the modular command pattern and provide genuine utility. The UUID generator serves as a reference implementation showing how to integrate Bubbletea TUI components (interactive list selection) with practical functionality that developers use daily. This validates the foundation architecture and creates a template for future commands.

## What Changes
- Add UUID generation utility as first module (`modules/uuid/`)
- Implement interactive selection using Bubbles list component
- Support UUID v4 (random) and v7 (time-ordered) generation
- Register module with root command using established pattern
- Add dependencies: `github.com/google/uuid`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/bubbletea`

## Impact
- **Affected specs**:
  - Creates new capability `uuid-generator`
  - Modifies `cli-foundation` to show module registration in practice
- **Affected code**:
  - New: `modules/uuid/cmd.go`, `modules/uuid/uuid.go`
  - Modified: `cmd/root.go` (adds module registration in init function)
- **Dependencies**:
  - Adds `github.com/google/uuid` for UUID generation
  - Adds `github.com/charmbracelet/bubbles` for list component
  - Adds `github.com/charmbracelet/bubbletea` for TUI framework
- **Testing**: Deliverable is `cly uuid` running interactively with list selection
