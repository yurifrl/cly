# Tasks: add-bundle-command

## Implementation Order

- [x] Create module skeleton with Register function and root command
- [x] Implement Bundler interface and common file/state parsing
- [x] Implement BrewBundler (calls brew bundle)
- [x] Implement GoBundler with mise detection
- [x] Implement JsBundler with GitHub shorthand normalization
- [x] Implement PythonBundler using uv tool
- [x] Add --edit, --no-edit, --dry-run, --file flags
- [x] Register module in cmd/root.go
- [x] Write tests for file parsing and state diff logic
- [x] Write integration tests per bundler type

## Parallelizable

Tasks 3-6 (individual bundlers) can be done in parallel after task 2 completes.

## Validation

- `cly bundle --help` shows usage
- `cly bundle brew --dry-run` shows diff without changes
- `cly bundle go` syncs Gofile packages
- Tests pass: `go test ./modules/bundle/...`
