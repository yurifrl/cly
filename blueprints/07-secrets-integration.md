# Blueprint: 1Password Secrets Integration

## Goal
Automatic 1Password secret resolution in config system using CLI authentication.

## Problem
Config values like API tokens need to be stored securely, not in plain text. Users already have 1Password with CLI integration - leverage it.

## Solution
Transparently resolve `op://` references in config during load.

## Approach

### Flow
```
Load() → viper.Unmarshal(&cfg) → resolveSecrets(ctx, &cfg.Modules) → cache
```

### Components

**SecretResolver interface**
```go
type SecretResolver interface {
    Resolve(ctx context.Context, ref string) (string, error)
}
```

**OpResolver**
- Executes `op read <uri>` via CLI
- URI: `op://vault-name/item-name/field-name`
- Uses existing app auth (no tokens needed)
- 10s timeout
- Clear errors

**resolveSecretsInPlace**
- Walk `map[string]interface{}` recursively
- Find `op://` strings
- Replace in-place with resolved values
- Fail fast on first error

### Integration
`pkg/config/config.go:81-83` - after `v.Unmarshal(&cfg)`, before caching

## Usage

```yaml
modules:
  api:
    token: op://Personal/github-token/credential
    url: https://api.github.com
```

On load, `token` becomes the actual secret value.

## Implementation (TDD)

### Phase 1: Core Resolver
- Test format validation
- Test CLI execution (mock binary)
- Test timeout handling
- Test error cases
- Implement OpResolver

### Phase 2: Walker
- Test flat map resolution
- Test nested structures
- Test mixed value types
- Test empty map
- Test fail-fast behavior
- Implement resolveSecretsInPlace

### Phase 3: Integration
- Test Load() without secrets (regression)
- Test Load() with secrets
- Test error propagation
- Wire into Load()
- Add example to default config

## Testing
Mock `op` binary in temp dir for tests - no shell execution in real tests.

## Edge Cases
- Empty modules → skip resolution
- Non-secrets → pass through
- CLI not found → clear error with hint
- Not authenticated → clear error
- Timeout → include which secret failed
- Partial resolution → never (fail fast)

## Files

**New:**
- `pkg/config/secrets.go`
- `pkg/config/secrets_test.go`

**Modified:**
- `pkg/config/config.go` - integration in Load()
- `config/config.yaml` - example with op:// reference

## Success
- All tests pass
- >80% coverage for secrets.go
- No secrets in errors/logs
- Zero overhead for configs without secrets
- Existing tests pass
