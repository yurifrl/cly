# OpenCode Notifications - Summary

## What This Does

Adds OpenCode plugin that sends desktop notifications when AI sessions become idle, showing the last assistant message as notification text.

## User Experience

```bash
# Install plugin (one command)
cly notify opencode install

# Restart OpenCode
# → Notifications appear automatically when tasks complete
```

## Architecture

```
OpenCode (TypeScript Plugin)
    ↓ session.idle event
    ↓ extract & clean last message
    ↓ call: cly notify hook opencode --message "..."
    ↓
CLY (Go Binary)
    ↓ read config.yaml
    ↓ check if enabled
    ↓ truncate message to length
    ↓ send notification
    → Desktop notification
    → Zellij notification (if enabled)
```

## Key Features

1. **Zero Configuration**: Single command install - no shell profile editing
2. **Configurable**: YAML config for title, sound, message length
3. **Smart Text**: Extracts & cleans last assistant message (removes code blocks)
4. **Reuses Infrastructure**: Leverages existing beeep, Zellij, sound system
5. **Fail Silently**: Plugin errors don't interrupt OpenCode

## Configuration

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ OpenCode Complete"
        message: "Task completed"
        sound: "Glass"
        message_length: 200     # max chars before truncation
```

## Files to Create

- `modules/notify/opencode/cmd.go` - Commands (install, status, uninstall)
- `modules/notify/opencode/install.go` - Installation logic
- `modules/notify/opencode/plugin/index.ts` - Plugin implementation

## Files to Modify

- `modules/notify/cmd.go` - Add --message flag with truncation, register opencode
- `modules/config/config.yaml` - Add opencode hook config
- `pkg/config/config.go` - Add MessageLength field

## Implementation Phases

1. **Phase 1**: TypeScript plugin (index.ts) - extract & clean message
2. **Phase 2**: Go installation (install.go, cmd.go) - embed & deploy plugin
3. **Phase 3**: Hook handler (cmd.go) - add --message flag, truncation
4. **Phase 4**: Config structure (YAML + Go structs) - add message_length field

## No Breaking Changes

- Existing notify hooks unchanged
- New feature, opt-in only
- All existing functionality preserved
- Rollback: uninstall plugin

## Testing (Manual)

- Install/uninstall
- Status command
- Direct hook test: `cly notify hook opencode --message "test"`
- OpenCode integration (wait for idle)
- Message truncation
- Sound variants
- Enable/disable

## Success Metrics

✅ Single command install
✅ Zero configuration (no env vars)
✅ Notifications work after restart
✅ Config options respected
✅ Silent failures (no OpenCode interruption)
✅ Reuses existing infrastructure
✅ Clear debugging commands
