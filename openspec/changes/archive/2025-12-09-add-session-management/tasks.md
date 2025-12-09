## 1. Session Package
- [x] Create `pkg/session/session.go` with Initialize function
- [x] Implement two-word name generator (adjectives + animals)
- [x] Add name validation (alphanumeric, hyphens, underscores)

## 2. Claude Module
- [x] Create `modules/claude/cmd.go` with `cly claude` command
- [x] Add `--name` flag (optional value)
- [x] Check `CLAUDE_SESSION_NAME` env var for default
- [x] Export `CLAUDE_SESSION_NAME` to child process
- [x] Execute `claude` with passthrough args

## 3. Zellij Integration
- [x] Detect Zellij via `$ZELLIJ` env var
- [x] Run `zellij action rename-tab <name>` when detected

## 4. Testing
- [x] Unit tests for name generation
- [x] Unit tests for name validation
- [x] Integration tests for env var handling
- [x] Integration tests for Zellij detection (mocked)
