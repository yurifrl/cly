# Design: Session Management

## Context
`cly claude` wraps Claude Code with session naming and Zellij tab integration. Users get memorable session names and visual tab context.

## Goals / Non-Goals

**Goals:**
- Wrap Claude Code with session context
- Auto-generate or accept explicit session names
- Update Zellij tab name to match session
- Export `CLAUDE_SESSION_NAME` for Claude Code

**Non-Goals:**
- Session persistence across restarts
- Support for tmux or other multiplexers
- Session history or resumption

## Decisions

### Decision: Two-Word Name Generation
Use adjective + animal format for auto-generated names.

**Rationale:**
- Human-memorable: `QuickFox`, `BrightOwl`
- Simple implementation, no external dependencies
- Collision probability acceptable for CLI scope

### Decision: Zellij via CLI Command
Use `zellij action rename-tab` instead of escape sequences.

**Rationale:**
- Documented, stable API
- Simpler than escape sequence handling
- Silent failure if not in Zellij

### Decision: Module Structure
New `modules/claude/` with session logic in `pkg/session/`.

**Rationale:**
- Follows existing module pattern
- Session logic reusable by other modules
- Clean separation of concerns

## Architecture

### Package Structure
```
modules/claude/
└── cmd.go          # cly claude command

pkg/session/
├── session.go      # Initialize, Session struct
├── generator.go    # Name generation
├── zellij.go       # Zellij integration
└── session_test.go
```

### Data Flow
1. Parse `--name` flag
2. Check `CLAUDE_SESSION_NAME` env if no flag
3. Generate name if needed
4. Validate name
5. Detect Zellij, rename tab if present
6. Print session indicator
7. Export env and exec `claude`

### Integration
```go
// modules/claude/cmd.go
var Cmd = &cobra.Command{
    Use:   "claude",
    Short: "Run Claude Code with session management",
    RunE: func(cmd *cobra.Command, args []string) error {
        name, _ := cmd.Flags().GetString("name")
        sess, err := session.Initialize(name)
        if err != nil {
            return err
        }
        fmt.Printf("🏷️  Session: %s\n", sess.Name)
        return sess.ExecClaude(args)
    },
}
```

## Risks / Trade-offs

### Risk: Name Collisions
Low probability with 50+ words in each pool (2500+ combinations).

### Trade-off: Zellij-Only
Non-Zellij users don't get tab integration. Feature degrades gracefully.
