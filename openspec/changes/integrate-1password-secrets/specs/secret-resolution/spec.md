# secret-resolution Specification

## Purpose
Automatically resolve 1Password secret references in configuration, allowing secure credential storage without hardcoding secrets in YAML files.

## ADDED Requirements

### Requirement: Secret Reference Format
The system SHALL recognize and resolve 1Password secret references in configuration.

#### Scenario: Valid secret reference format
- **WHEN** config contains string `op://vault-name/item-name/field-name`
- **THEN** system SHALL recognize it as secret reference
- **AND** resolve via `op read` CLI command
- **AND** replace with actual secret value

#### Scenario: Invalid reference format rejected
- **WHEN** config contains string missing `op://` prefix
- **OR** path has fewer than 3 components (vault/item/field)
- **THEN** resolution SHALL fail with clear error
- **AND** error SHALL indicate expected format

#### Scenario: Non-secret strings preserved
- **WHEN** config contains `https://op://example.com`
- **OR** string contains `op://` but not at start
- **THEN** value SHALL pass through unchanged
- **AND** no resolution attempted

### Requirement: Automatic Resolution
The system SHALL automatically resolve secrets during config load.

#### Scenario: Secrets resolved in Load()
- **WHEN** `config.Load()` is called
- **AND** config contains `op://` references in `modules.*`
- **THEN** secrets SHALL be resolved before returning config
- **AND** modules receive actual secret values via `GetString()`

#### Scenario: Resolution timing
- **WHEN** config is loaded
- **THEN** secrets SHALL be resolved once at load time
- **AND** cached with resolved values
- **AND** not re-resolved on each `Get()` call

#### Scenario: Modules sections only
- **WHEN** config contains `op://` in `app`, `theme`, or `bundle` sections
- **THEN** these SHALL NOT be resolved (only `modules.*` resolved)
- **AND** pass through as literal strings

### Requirement: CLI-Based Resolution
The system SHALL use 1Password CLI for secret resolution.

#### Scenario: Execute op CLI
- **WHEN** resolving `op://vault/item/field`
- **THEN** system SHALL execute `op read op://vault/item/field`
- **AND** capture stdout as secret value
- **AND** trim whitespace from result

#### Scenario: Use existing authentication
- **WHEN** op CLI is executed
- **THEN** it SHALL use existing desktop app authentication
- **AND** respect biometric unlock settings
- **AND** not require service account tokens

#### Scenario: CLI not found
- **WHEN** `op` binary not in PATH
- **THEN** resolution SHALL fail with error
- **AND** error SHALL suggest installing 1Password CLI
- **AND** include link to installation docs

### Requirement: Timeout Handling
The system SHALL enforce timeouts on secret resolution.

#### Scenario: Default timeout
- **WHEN** resolving secrets
- **THEN** system SHALL use 10 second timeout
- **AND** timeout SHALL apply per-secret (not total)

#### Scenario: Timeout exceeded
- **WHEN** `op read` takes longer than timeout
- **THEN** operation SHALL be cancelled
- **AND** error SHALL indicate timeout occurred
- **AND** error SHALL include which secret timed out

#### Scenario: Context cancellation
- **WHEN** parent context is cancelled
- **THEN** secret resolution SHALL stop immediately
- **AND** return context cancellation error

### Requirement: Error Handling
The system SHALL fail fast on secret resolution errors.

#### Scenario: Resolution failure stops load
- **WHEN** any secret fails to resolve
- **THEN** `config.Load()` SHALL return error
- **AND** app SHALL not start with unresolved secrets
- **AND** no partial resolution occurs

#### Scenario: Clear error messages
- **WHEN** resolution fails
- **THEN** error SHALL indicate which config key failed
- **AND** include underlying error from `op` CLI
- **AND** not leak actual secret values in error

#### Scenario: Authentication failure
- **WHEN** `op` CLI returns authentication error
- **THEN** error SHALL indicate user not signed in
- **AND** suggest running `op signin` or enabling app integration

#### Scenario: Secret not found
- **WHEN** `op` CLI cannot find vault/item/field
- **THEN** error SHALL indicate secret reference is invalid
- **AND** include the reference that failed
- **AND** suggest checking vault/item/field names

### Requirement: Recursive Resolution
The system SHALL resolve secrets in nested configuration structures.

#### Scenario: Flat map resolution
- **WHEN** config has `modules.api.token: op://vault/item/field`
- **THEN** `token` value SHALL be resolved
- **AND** other keys in `api` unchanged

#### Scenario: Nested map resolution
- **WHEN** config has nested maps under `modules.*`
- **THEN** system SHALL recursively walk all levels
- **AND** resolve secrets at any depth
- **AND** preserve non-secret values at all levels

#### Scenario: Mixed value types
- **WHEN** config contains strings, ints, bools, floats
- **THEN** only string values starting with `op://` resolved
- **AND** other types pass through unchanged
- **AND** type safety preserved

#### Scenario: Empty maps handled
- **WHEN** `modules` is empty map
- **OR** module section is empty
- **THEN** resolution SHALL succeed with no errors
- **AND** no resolution attempted

### Requirement: Security
The system SHALL handle secrets securely.

#### Scenario: No secret logging
- **WHEN** secrets are resolved
- **THEN** actual secret values SHALL NOT be logged
- **AND** only `op://` references logged (if any)

#### Scenario: No secret caching between loads
- **WHEN** `Load()` is called multiple times
- **THEN** secrets SHALL be re-resolved each time
- **AND** not persisted to disk
- **AND** not cached in memory between calls

#### Scenario: Error messages sanitized
- **WHEN** resolution errors occur
- **THEN** error messages SHALL NOT contain secret values
- **AND** only contain `op://` references
- **AND** safe to display to user or log

### Requirement: Backward Compatibility
The system SHALL not break existing configs without secrets.

#### Scenario: Configs without secrets work
- **WHEN** config contains no `op://` references
- **THEN** `Load()` SHALL work unchanged
- **AND** no performance overhead
- **AND** no additional dependencies required

#### Scenario: Empty modules section
- **WHEN** config has no `modules` section
- **OR** `modules` is empty map
- **THEN** `Load()` SHALL succeed
- **AND** skip secret resolution

#### Scenario: Existing tests pass
- **WHEN** change is implemented
- **THEN** all existing config tests SHALL still pass
- **AND** no regressions in config loading
