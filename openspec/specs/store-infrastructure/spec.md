# store-infrastructure Specification

## Purpose
TBD - created by archiving change add-bundle-command. Update Purpose after archive.
## Requirements
### Requirement: Store Interface

The store SHALL provide a generic namespace/key interface for state persistence.

#### Scenario: List returns keys in namespace

- **WHEN** calling `List(namespace)`
- **THEN** SHALL return all keys in that namespace
- **AND** empty slice if namespace has no keys

#### Scenario: Add inserts key

- **WHEN** calling `Add(namespace, key)`
- **THEN** key SHALL be persisted in namespace
- **AND** duplicate add SHALL be idempotent (no error)

#### Scenario: Remove deletes key

- **WHEN** calling `Remove(namespace, key)`
- **THEN** key SHALL be removed from namespace
- **AND** removing non-existent key SHALL be idempotent (no error)

### Requirement: Database Location

The store SHALL use a shared DuckDB database.

#### Scenario: Default location

- **WHEN** Store is created with default path
- **THEN** database SHALL be at `~/.config/cly/cly.db`
- **AND** directory SHALL be created if missing

#### Scenario: Database created on first use

- **WHEN** database file does not exist
- **THEN** Store SHALL create it on first operation
- **AND** schema SHALL be auto-migrated

### Requirement: Injection Pattern

The store SHALL be injected into modules that need it.

#### Scenario: Module receives store

- **WHEN** module needs state persistence
- **THEN** Store SHALL be passed via Register function
- **AND** module SHALL not create its own Store instance

