# Tasks: add-bundle-command

## Implementation Order

- [ ] Create module skeleton with Register function and root command
- [ ] Implement Bundler interface and common file/state parsing
- [ ] Implement BrewBundler (calls brew bundle)
- [ ] Implement GoBundler with mise detection
- [ ] Implement JsBundler with GitHub shorthand normalization
- [ ] Implement PythonBundler using uv tool
- [ ] Add --edit, --no-edit, --dry-run, --file flags
- [ ] Register module in cmd/root.go
- [ ] Write tests for file parsing and state diff logic
- [ ] Write integration tests per bundler type

## Parallelizable

Tasks 3-6 (individual bundlers) can be done in parallel after task 2 completes.

## Validation

- `cly bundle --help` shows usage
- `cly bundle brew --dry-run` shows diff without changes
- `cly bundle go` syncs Gofile packages
- Tests pass: `go test ./modules/bundle/...`
