# Tasks: Integrate 1Password Secrets

## Phase 1: Core Resolver (TDD)

### Task 1.1: Write OpResolver format validation tests
- [x] Test valid `op://vault/item/field` format accepted
- [x] Test missing `op://` prefix rejected
- [x] Test incomplete paths rejected (missing vault/item/field)
- [x] Test empty strings rejected
- **Validation**: ✅ Tests pass

### Task 1.2: Write OpResolver CLI execution tests
- [x] Create mock `op` binary in temp dir
- [x] Test successful resolution with mock binary
- [x] Test mock returns correct secret value
- [x] Test OpResolver uses custom CLI path
- **Validation**: ✅ Tests pass

### Task 1.3: Write OpResolver timeout tests
- [x] Test context timeout propagates to CLI execution
- [x] Test timeout includes which secret failed
- [x] Test 10s default timeout
- **Validation**: ✅ Tests pass

### Task 1.4: Write OpResolver error tests
- [x] Test CLI not found error
- [x] Test CLI authentication failure
- [x] Test secret not found error
- [x] Test error messages are clear (no secret values leaked)
- **Validation**: ✅ Tests pass

### Task 1.5: Implement OpResolver
- [x] Create `pkg/config/secrets.go`
- [x] Implement `SecretResolver` interface
- [x] Implement `OpResolver` struct with `Resolve(ctx, ref)` method
- [x] Validate `op://` format
- [x] Execute `op read <ref>` with context timeout
- [x] Return clear errors
- **Validation**: ✅ All Phase 1 tests pass

## Phase 2: Recursive Walker (TDD)

### Task 2.1: Write flat map resolution test
- [x] Test single secret in flat map resolves
- [x] Test non-secret values pass through unchanged
- **Validation**: ✅ Test passes

### Task 2.2: Write nested map resolution test
- [x] Test secrets in nested `map[string]interface{}` resolve
- [x] Test multiple levels of nesting
- **Validation**: ✅ Test passes

### Task 2.3: Write mixed value types test
- [x] Test secrets resolve alongside strings, ints, bools, floats
- [x] Test non-string values unchanged
- [x] Test arrays/slices ignored
- **Validation**: ✅ Test passes

### Task 2.4: Write empty map test
- [x] Test empty map causes no errors
- [x] Test nil values handled safely
- **Validation**: ✅ Test passes

### Task 2.5: Write fail-fast test
- [x] Test first error stops processing
- [x] Test error includes which key failed
- [x] Test partial resolution never happens
- **Validation**: ✅ Test passes

### Task 2.6: Write non-secret string test
- [x] Test strings containing `op://` but not at start pass through
- [x] Test URLs like `https://op://example.com` unchanged
- **Validation**: ✅ Test passes

### Task 2.7: Implement resolveSecretsInPlace
- [x] Add `resolveSecretsInPlace(ctx, resolver, data)` function to `secrets.go`
- [x] Recursively walk `map[string]interface{}`
- [x] Detect strings starting with `op://`
- [x] Call resolver.Resolve() for secrets
- [x] Replace in-place with resolved value
- [x] Return first error (fail-fast)
- **Validation**: ✅ All Phase 2 tests pass

## Phase 3: Integration (TDD)

### Task 3.1: Write Load() regression test
- [x] Test Load() without secrets works unchanged
- [x] Test existing config structure preserved
- [x] Test no performance regression
- **Validation**: ✅ Test passes (baseline)

### Task 3.2: Write Load() with secrets test
- [x] Create temp config with `op://` references
- [x] Mock op CLI in test
- [x] Test secrets resolved to actual values
- [x] Test module code receives resolved secrets via GetString()
- **Validation**: ✅ Test passes

### Task 3.3: Write Load() error propagation test
- [x] Test Load() returns error when secret resolution fails
- [x] Test error message is clear
- [x] Test no secrets leaked in error
- **Validation**: ✅ Test passes

### Task 3.4: Integrate into config.Load()
- [x] Modify `pkg/config/config.go`
- [x] Add context creation with 10s timeout after `v.Unmarshal(&cfg)`
- [x] Call `resolveSecretsInPlace(ctx, NewOpResolver(), cfg.Modules)`
- [x] Return error if resolution fails
- **Validation**: ✅ All Phase 3 tests pass

### Task 3.5: Update default config with example
- [x] Modify `defaultConfig` in `config.go`
- [x] Add commented example showing `op://` usage
- [x] Document in comment: vault/item/field format
- **Validation**: ✅ Default config includes example, `go build` succeeds

## Phase 4: Validation

### Task 4.1: Run full test suite
- [x] Execute `go test ./pkg/config/...`
- [x] Verify all tests pass
- [x] Verify no secrets in test output
- **Validation**: ✅ Exit code 0, 17 tests pass

### Task 4.2: Run coverage report
- [x] Execute `go test -cover ./pkg/config/`
- [x] Verify `secrets.go` coverage >80%
- **Validation**: ✅ Coverage: secrets.go 92.9%

### Task 4.3: Manual verification
- [x] Build binary: `go build`
- **Validation**: ✅ Build succeeds

### Task 4.4: Error path verification
- [x] Test with `op` CLI not installed
- [x] Test with 1Password not authenticated
- [x] Test with invalid `op://` reference
- [x] Verify clear error messages
- **Validation**: ✅ Errors are actionable, no panics (covered by tests)

## Dependencies
- Task 1.5 depends on 1.1-1.4 (TDD)
- Task 2.7 depends on 2.1-2.6 (TDD)
- Task 3.4 depends on 1.5 and 2.7 (implementation complete)
- Phase 4 depends on all previous phases

## Parallelizable Work
- Phase 1 and Phase 2 tests can be written in parallel
- Phase 1 and Phase 2 implementation must be sequential (resolver before walker)
