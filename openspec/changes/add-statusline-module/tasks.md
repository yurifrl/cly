## 1. Core
- [ ] 1.1 Create `types.go` - StatusJSON, Config structs
- [ ] 1.2 Write context tests
- [ ] 1.3 Implement `context.go` - token calc, color by threshold
- [ ] 1.4 Write cmd tests (subcommands + format string)
- [ ] 1.5 Implement `cmd.go` - main cmd + subcommands + format parsing
- [ ] 1.6 Implement custom command with timeout
- [ ] 1.7 Add config struct to `pkg/config/`

## 2. Integration
- [ ] 2.1 Write integration test - JSON in → output check
- [ ] 2.2 Register in `cmd/root.go`

## 3. Verify
- [ ] 3.1 Run tests
- [ ] 3.2 Manual: `echo '{"context_window":...}' | cly statusline`
