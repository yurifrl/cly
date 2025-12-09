# Change: Session Management

## Why
CLI needs named sessions for better user experience. Currently sessions are identified only by random UUIDs, making it difficult for users to track and resume specific work contexts across terminal sessions.

## What Changes
- Named session capability: Allow users to specify memorable session names
- Auto-generated session names: Two-word random format when no name provided
- Terminal integration: Update Zellij tab/pane names to match session
- Environment variable exports: Export `CLY_SESSION_NAME` for downstream tools

## Impact
- Affected specs: Creates new `session-management` capability
- Affected code:
  - `cmd/root.go` - Add session initialization in PreRun
  - New `pkg/session/` package for session logic
  - Integration with terminal (Zellij only)
