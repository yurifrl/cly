# Change: Add Demo Namespace with Chat Example

## Why
CLY needs a `demo` namespace to showcase Charm components without mixing them with utility commands. The chat example from Bubbletea demonstrates textarea and viewport components working together - essential patterns for building interactive TUIs. This separates educational demos from production utilities, keeping the command structure clean.

## What Changes
- Add `demo` parent command as a namespace for component showcases
- Implement `demo chat` subcommand adapted from `references/bubbletea/examples/chat`
- Chat demo combines textarea (input) and viewport (scrollable message history)
- Establish pattern for future demo subcommands (spinner, table, list, etc.)

## Impact
- **Affected specs**:
  - Creates new capability `demo-commands`
  - Modifies `cli-foundation` to document namespace pattern
- **Affected code**:
  - New: `modules/demo/cmd.go` (parent namespace command)
  - New: `modules/demo/chat/cmd.go`, `modules/demo/chat/chat.go`
  - Modified: `cmd/root.go` (register demo namespace)
- **Dependencies**:
  - Adds `github.com/charmbracelet/bubbles/textarea` (new component)
  - Already has: bubbles/viewport, bubbletea, lipgloss
- **Testing**: Deliverable is `cly demo chat` running interactive chat UI
