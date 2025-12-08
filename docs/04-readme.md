# Phase 4: README

**Goal**: Document what you built

**Deliverable**: README.md

**Time**: 15 minutes

---

## Create README.md


**File: `README.md`**
```markdown
# CLY - Command Line Utilities

Day-to-day utility CLI built with Charm libraries.

## Installation

```bash
go install github.com/yourusername/cly@latest
```

Or build from source:
```bash
git clone https://github.com/yourusername/cly
cd cly
go build
```

## Usage

```bash
# Show all commands
cly --help
cly ls

# Generate UUID
cly uuid

# Format JSON
echo '{"name":"test"}' | cly json

# Encode base64
echo "hello" | cly encode base64
```

## Commands

| Command | Description |
|---------|-------------|
| `uuid` | Generate UUIDs (v4, v7, multiple) |
| `json` | Format and validate JSON |
| `encode` | Base64/URL/hex encoding |
| `http` | Quick HTTP requests |
| `version` | Show version |
| `ls` | List all commands |

## Architecture

Modular design with clean command registration:
- Each command is a self-contained module
- Zero coupling between modules
- Easy to add utilities

```
modules/
├── uuid/
│   ├── cmd.go       # Registration
│   └── uuid.go      # Implementation
└── json/
    ├── cmd.go
    └── json.go
```

## Adding a Utility

See `docs/module-template.md` for template.

- Copy existing module
- Adapt for new utility
- Register in `cmd/root.go`

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling

## License

MIT
```

---

## Success Criteria

✅ README.md exists
✅ Documents all commands
✅ Installation instructions clear
✅ Usage examples included

---

## Next Steps

**Optional polish** (do later as needed):
- Add `cly version` command
- Add `cly ls` command to list utilities
- Refactor large modules to MVC pattern (model/view/update)
- Add tests
- Build and distribute binary
