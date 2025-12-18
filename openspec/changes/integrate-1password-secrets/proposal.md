# Proposal: Integrate 1Password Secrets

## Problem
Configuration values like API tokens, credentials, and secrets need secure storage. Storing them in plain text YAML files is insecure. Users already have 1Password with CLI integration enabled.

## Solution
Transparently resolve `op://` secret references in config during load time. When config contains `op://vault/item/field`, automatically execute `op read` and replace with the actual secret value.

## User Experience

### Before (insecure)
```yaml
modules:
  backup:
    gcs_token: "ya29.a0AfH6SMBx..." # plain text secret
```

### After (secure)
```yaml
modules:
  backup:
    gcs_token: op://Personal/gcs-backup/token
```

Module code unchanged:
```go
token := config.GetString("modules.backup.gcs_token")
// Gets actual secret, not op:// reference
```

## Approach

### Architecture
- **SecretResolver interface** - abstraction for resolution
- **OpResolver** - implements resolution via `op read` CLI
- **resolveSecretsInPlace** - recursive walker for config maps
- **Integration point** - `config.Load()` after Viper unmarshal

### Resolution Flow
```
Load()
  → viper.ReadInConfig()
  → viper.Unmarshal(&cfg)
  → resolveSecretsInPlace(ctx, &cfg.Modules)  ← NEW
  → return cfg (with resolved secrets)
```

### Authentication
Uses existing 1Password desktop app integration (Settings > Developer > Integrate with 1Password CLI). No service account tokens needed.

## Scope

### In Scope
- Automatic resolution of `op://` references in `modules.*` config
- CLI-based resolution using `op read`
- Context timeout (10s default)
- Fail-fast error handling
- TDD implementation with >80% coverage

### Out of Scope
- Secret caching between loads
- Other secret providers (Vault, AWS Secrets Manager)
- Secrets in non-module config sections (app, theme, bundle)
- Secret encryption at rest
- Secret rotation/refresh

## Impact

### Benefits
- Secure credential management without changing module code
- Zero new dependencies (uses existing `op` CLI)
- Transparent to modules - works with existing `config.GetString()`
- No performance impact for configs without secrets

### Risks
- Requires 1Password CLI installed and authenticated
- Adds latency during config load (network call to 1Password)
- Failed resolution blocks app startup (fail-fast design)

### Migration
- Existing configs without secrets: no change required
- Configs with secrets: replace plain text with `op://` references
- No breaking changes to config structure or API

## Implementation Strategy

### TDD Phases
1. **Core resolver** - OpResolver with format validation, CLI execution, error handling
2. **Walker** - resolveSecretsInPlace with recursive traversal, mixed types
3. **Integration** - Wire into Load(), regression tests, examples

### Testing Approach
- Mock `op` binary in temp dir for isolated tests
- No real 1Password calls in CI
- Test coverage for all edge cases (timeout, auth failure, missing CLI)

## Dependencies
- No new Go dependencies
- Runtime dependency: 1Password CLI (`op`) installed and authenticated

## Alternatives Considered

### 1. Service Account Token (SDK)
- **Rejected**: Requires separate service account setup, different credentials from user's 1Password

### 2. Lazy Resolution on Get()
- **Rejected**: Adds complexity, caching inconsistencies, secrets resolved multiple times

### 3. Separate LoadWithSecrets() Function
- **Rejected**: API proliferation, requires all callers to opt-in

### 4. Support Multiple Secret Providers
- **Rejected**: YAGNI - can add later if needed, start with 1Password only

## Success Criteria
- All tests pass with >80% coverage
- Zero performance impact for configs without secrets
- No secrets leaked in error messages or logs
- Existing config tests still pass (no regressions)
- Clear error messages when CLI unavailable or auth fails
