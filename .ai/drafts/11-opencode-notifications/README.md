# OpenCode Notifications Integration

Complete draft for adding OpenCode notification support to cly.

## Draft Documents

1. **[draft.md](./draft.md)** - Complete technical specification
   - Architecture overview
   - Implementation plan (4 phases)
   - File-by-file changes
   - Configuration structure
   - Testing strategy
   - ~800 lines

2. **[summary.md](./summary.md)** - Executive summary
   - High-level overview
   - Key features
   - Quick reference
   - ~80 lines

3. **[checklist.md](./checklist.md)** - Implementation checklist
   - Step-by-step tasks
   - Testing verification
   - Build checks
   - ~200 lines

4. **[faq.md](./faq.md)** - Frequently asked questions
   - Installation questions
   - Configuration help
   - Troubleshooting
   - ~150 questions

## Quick Start (for implementer)

1. Read [summary.md](./summary.md) for overview
2. Read [draft.md](./draft.md) for complete specification
3. Use [checklist.md](./checklist.md) while implementing
4. Reference [faq.md](./faq.md) for edge cases

## Key Decisions

- **No npm publishing**: Plugin embedded in cly binary
- **No env vars**: Plugin calls `cly` commands directly via `$` shell API
- **Reuse infrastructure**: Leverage existing beeep/Zellij/sound system
- **Fail silently**: Plugin errors don't interrupt OpenCode
- **200 char messages**: Balance detail vs readability (configurable)

## Implementation Phases

1. **Phase 1**: TypeScript plugin (index.ts) - extract & clean message text
2. **Phase 2**: Go installation (install.go, cmd.go) - embed & deploy
3. **Phase 3**: Hook handler (add --message flag, truncation logic)
4. **Phase 4**: Configuration (YAML + Go structs, message_length field)

## Files Created

```
modules/notify/opencode/
├── cmd.go                  # Commands: install, uninstall, status
├── install.go              # Installation logic
└── plugin/
    └── index.ts            # Plugin implementation
```

## Files Modified

```
modules/notify/cmd.go       # Add --message flag, register opencode
modules/config/config.yaml  # Add opencode hook
pkg/config/config.go        # Add MessageLength field
```

## User Flow

```bash
# Install (one command - done!)
cly notify opencode install

# Restart OpenCode
# Notifications appear automatically when tasks complete
```

## Testing

Manual testing only (no automated tests requested):
- Installation/uninstallation
- Status command verification
- Direct hook testing
- OpenCode integration
- Configuration options
- Edge cases

## Success Criteria

✅ Single command install
✅ Zero configuration steps
✅ Notifications work after restart
✅ Config options respected
✅ Silent failures (no OpenCode interruption)
✅ Reuses existing infrastructure
✅ Clear debugging commands

## No Breaking Changes

- Existing notify hooks unchanged
- New feature, opt-in only
- All existing functionality preserved
- Clean rollback path

## Next Steps

1. Review draft documents
2. Answer any questions/clarifications
3. Proceed with implementation using checklist
4. Manual testing per testing plan
5. Documentation updates

---

**Draft Status**: ✅ Complete - Ready for review

**Version**: v2 (simplified, no environment variables)

**Last Updated**: 2025-12-28
