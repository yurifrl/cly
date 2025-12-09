# Design: Session Management

## Context
CLI currently has no session identity beyond random UUIDs. Users working across multiple terminal tabs/panes need human-readable session names to track work contexts. This feature adds named sessions with terminal integration.

## Goals / Non-Goals

**Goals:**
- Provide memorable session names (auto-generated or explicit)
- Export session context to environment for downstream tools
- Integrate with Zellij terminal for visual feedback

**Non-Goals:**
- Session persistence across restarts (future enhancement)
- Session history or resumption
- Multi-terminal multiplexer support beyond Zellij

## Decisions

### Decision: Two-Word Name Generation
Use simple random word combination (color/adjective + animal/noun) for auto-generated names.

**Rationale:**
- Human-memorable without cognitive load
- Collision probability acceptably low for CLI tool scope
- Simple implementation without external dependencies
- Examples: `QuickTask`, `BrightIdea`, `TempWork`

**Alternatives considered:**
- UUID prefixes: Not human-friendly
- Timestamp-based: Not memorable
- Word + number: Less memorable than word pairs

### Decision: Zellij-Only Terminal Integration
Support only Zellij for tab/pane name updates initially.

**Rationale:**
- Zellij provides documented escape sequences
- User base primarily uses Zellij (based on specs/04)
- Other multiplexers have inconsistent or undocumented APIs
- Can expand to tmux/others in future if needed

### Decision: Environment Variable Export Only
Export session name via env vars, don't manage session files or state.

**Rationale:**
- Simplest implementation for MVP
- Follows Unix philosophy (env vars for context)
- No file system dependencies or permissions issues
- Child processes inherit session context naturally

### Decision: Initialize in Root Command PreRun
Hook session initialization into Cobra's `PersistentPreRunE`.

**Rationale:**
- Executes before all commands automatically
- Proper error handling with early exit
- Consistent behavior across all subcommands
- Minimal code changes to existing commands

## Architecture

### Package Structure
```
pkg/session/
├── session.go      # Core session logic
├── generator.go    # Name generation
├── terminal.go     # Zellij integration
└── session_test.go # Tests
```

### Data Flow
1. Parse CLI flags (`--name`)
2. Check environment (`CLY_SESSION_NAME`)
3. Generate name if needed
4. Validate name
5. Export to environment
6. Detect terminal type
7. Update terminal (if Zellij)
8. Print session indicator

### Integration Point
```go
// cmd/root.go
RootCmd = &cobra.Command{
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Existing config load
        _, err := pkgconfig.Load()
        if err != nil {
            return err
        }

        // New session initialization
        name := cmd.Flag("name").Value.String()
        sess, err := session.Initialize(name)
        if err != nil {
            return err
        }
        fmt.Printf("🏷️  Session: %s\n", sess.Name)
        return nil
    },
}
```

## Risks / Trade-offs

### Risk: Name Collisions
Auto-generated names may collide if user runs many concurrent sessions.

**Mitigation:**
- Large word pools (50+ words each) = 2500+ combinations
- Session lifetime is typically short for CLI tools
- Collisions are cosmetic (no data corruption)

### Trade-off: Zellij-Only Support
Non-Zellij users won't get terminal integration.

**Mitigation:**
- Feature degrades gracefully (session still works, just no tab names)
- Can add tmux/others in future
- Document limitation clearly

### Risk: Environment Variable Pollution
Exporting env vars affects child processes.

**Mitigation:**
- Use clear `CLY_` prefix to avoid conflicts
- Document exported vars
- Standard practice for CLI context propagation

## Migration Plan
No migration needed - this is a new feature with no breaking changes.

## Open Questions
None - design is straightforward and scoped for initial implementation.
