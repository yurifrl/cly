# CLY - Overview

## What is CLY?

A modular Go CLI for **day-to-day utility tasks**, built with Charm libraries so you have working reference examples when implementing real features.

**Real utilities, beautiful TUI:**
```bash
cly uuid             # Generate UUIDs (uses textinput)
cly json             # Pretty-print/validate JSON (uses viewport)
cly http             # Quick HTTP requests (uses table for response)
cly encode           # Base64/URL encoding (uses form)
cly file-pick        # Interactive file picker
```

**Why Charm libraries?** When you need to add TUI features (forms, lists, tables) to your real commands, you have working reference implementations.

---

## Architecture

### Modular Pattern (from NSX-CLI)

```
cly/
├── main.go              # Entry point (10 lines)
├── cmd/
│   └── root.go          # Root command, registers all modules
├── modules/             # Each module = self-contained command
│   ├── spinner/
│   │   ├── cmd.go       # Register(parent) function
│   │   └── spinner.go   # Bubbletea implementation (Phase 2-3)
│   │                    # OR model.go/view.go/update.go (Phase 4)
│   └── <others>/
└── pkg/                 # Shared utilities
    └── style/
        └── theme.go     # Lipgloss styles
```

---

## Key Principle: Command Registration

**Single registration point** (`cmd/root.go`):
```go
func init() {
    spinner.Register(RootCmd)
    textinput.Register(RootCmd)
    list.Register(RootCmd)
    table.Register(RootCmd)
}
```

**Each module registers itself** (`modules/spinner/cmd.go`):
```go
func Register(parent *cobra.Command) {
    parent.AddCommand(&cobra.Command{
        Use:   "spinner",
        Short: "Animated spinner showcase",
        RunE:  run,
    })
}
```

**Benefits**:
- Adding command = 1 line in `cmd/root.go`
- Modules don't know about each other (zero coupling)
- Easy to remove/disable commands

---

## Reference Material

Use working examples from `references/bubbletea/examples/`:
- `spinner` → `references/bubbletea/examples/spinner`
- `textinput` → `references/bubbletea/examples/textinput`
- `list` → `references/bubbletea/examples/list-simple`
- `table` → `references/bubbletea/examples/table`

**Don't reinvent** - adapt working code.

---

## Implementation Phases

### Phase 1: Foundation (15 min)
**Deliverable**: `cly --help` works with styled output

**Files**: `main.go`, `go.mod`, `cmd/root.go`, `pkg/style/theme.go`

---

### Phase 2: First Utility (30 min)
**Deliverable**: `cly uuid` generates UUIDs with interactive options

**Why UUID first?** Simple utility that needs textinput (type selection) - real use case, showcases Charm component.

**Files**: `modules/uuid/cmd.go`, `modules/uuid/uuid.go`

---

### Phase 3: More Utilities (ongoing)
**Deliverable**: Real commands you'll actually use

**Pattern**: Each utility uses appropriate Charm components
- `json` - Uses viewport for scrolling output
- `http` - Uses table for response display
- `encode` - Uses form for input collection

---

### Phase 4: Polish (1.5 hours)
**Deliverable**: Production-ready

**Tasks**: Refactor to MVC when patterns emerge, add version/help, create README

---

## Success Criteria

**Phase 1**: ✅ `cly --help` shows beautiful output
**Phase 2**: ✅ `cly uuid` generates UUIDs (real utility, uses Charm libs)
**Phase 3**: ✅ Have 3-5 utilities you actually use daily
**Phase 4**: ✅ Ready to share, clean patterns

---

## Key Point

This is NOT a demo app. It's a **real utility CLI** where:
- Commands do useful work
- Implementation uses Charm libraries (so you have reference code)
- Architecture supports adding utilities forever (100+ commands)

When you need to add forms/tables/lists to new commands, you reference existing implementations.

---

## Next Steps

Read phase documentation in order:
1. `01-foundation.md` - Build minimal working CLI
2. `02-first-utility.md` - Add UUID generator (real utility)
3. `03-add-utilities.md` - Add more utilities as needed
4. `04-polish.md` - Production polish
