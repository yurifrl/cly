## 1. Core Session Package
- [ ] 1.1 Create `pkg/session/session.go` with name generation logic
- [ ] 1.2 Implement two-word random name generator (color + animal pools)
- [ ] 1.3 Add session name validation (alphanumeric, hyphens, underscores)
- [ ] 1.4 Export environment variables (`CLY_SESSION_NAME`)

## 2. CLI Integration
- [ ] 2.1 Add `--name` flag to root command
- [ ] 2.2 Check `CLY_SESSION_NAME` environment variable
- [ ] 2.3 Initialize session in `PersistentPreRunE`
- [ ] 2.4 Print session name with emoji indicator

## 3. Terminal Integration
- [ ] 3.1 Detect Zellij environment
- [ ] 3.2 Update tab name using Zellij escape sequences
- [ ] 3.3 Update pane name using Zellij escape sequences

## 4. Testing
- [ ] 4.1 Unit tests for name generation
- [ ] 4.2 Unit tests for environment variable detection
- [ ] 4.3 Integration tests for CLI flags
- [ ] 4.4 Integration tests for terminal detection
