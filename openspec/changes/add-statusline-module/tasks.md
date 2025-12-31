## 1. Core
- [x] 1.1 Create `types.go` - StatusJSON, Config structs
- [x] 1.2 Write context tests
- [x] 1.3 Implement `context.go` - token calc, color by threshold
- [x] 1.4 Write cmd tests (subcommands + format string)
- [x] 1.5 Implement `cmd.go` - main cmd + subcommands + format parsing
- [x] 1.6 Implement custom command with timeout
- [x] 1.7 Add config struct to `pkg/config/`

## 2. Integration
- [x] 2.1 Write integration test - JSON in → output check
- [x] 2.2 Register in `cmd/root.go`

## 3. Verify
- [x] 3.1 Run tests
- [x] 3.2 Manual: `echo '{"context_window":...}' | cly statusline`
