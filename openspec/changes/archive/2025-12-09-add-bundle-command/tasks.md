# Tasks: add-bundle-command

## Implementation Order

- [x] Implement pkg/store with Store interface and DuckDB implementation
- [x] Write tests for pkg/store (Add, Remove, List)
- [x] Create bundle module skeleton with Register(root, store)
- [x] Implement Bundler interface and common file parsing
- [x] Implement BrewBundler (delegates to brew bundle)
- [x] Implement GoBundler with mise detection
- [x] Implement JsBundler with GitHub shorthand normalization
- [x] Implement PythonBundler using uv tool
- [x] Add check subcommand
- [x] Add cleanup subcommand
- [x] Register module in cmd/root.go with Store injection
- [x] Write integration tests per bundler type

## Parallelizable

Tasks 5-8 (individual bundlers) can run in parallel after task 4.

## Validation

- [x] `cly bundle --help` shows subcommands and types
- [x] `go test ./pkg/store/...` passes
- [x] `go test ./modules/bundle/...` passes
