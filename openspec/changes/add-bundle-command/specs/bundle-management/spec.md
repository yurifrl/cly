# bundle-management Specification

## Purpose

Declarative package management for multiple ecosystems (brew, go, js, python) through a unified CLI interface.

## Requirements

### Requirement: Bundle Command Structure

The bundle command SHALL provide a unified interface for managing packages across ecosystems.

#### Scenario: Default type is brew

- **WHEN** user runs `cly bundle` without arguments
- **THEN** command SHALL execute brew bundle sync
- **AND** equivalent to running `cly bundle brew`

#### Scenario: Type argument selects bundler

- **WHEN** user runs `cly bundle <type>` where type is brew|go|js|python
- **THEN** command SHALL dispatch to appropriate bundler
- **AND** invalid type SHALL produce error with valid options

#### Scenario: Help shows available types

- **WHEN** user runs `cly bundle --help`
- **THEN** output SHALL list available bundle types
- **AND** show flag documentation

### Requirement: Editor Integration

The bundle command SHALL optionally open the bundle file in editor before sync.

#### Scenario: Edit enabled by default

- **WHEN** user runs `cly bundle <type>` without --no-edit
- **THEN** command SHALL open bundle file in $EDITOR
- **AND** wait for editor to close before syncing

#### Scenario: No-edit skips editor

- **WHEN** user runs `cly bundle <type> --no-edit`
- **THEN** command SHALL proceed directly to sync
- **AND** not open any editor

### Requirement: Dry Run Mode

The bundle command SHALL support previewing changes without executing them.

#### Scenario: Dry run shows diff

- **WHEN** user runs `cly bundle <type> --dry-run`
- **THEN** command SHALL show packages to install
- **AND** show packages to remove
- **AND** not execute any install/uninstall operations
- **AND** not modify state file

### Requirement: Custom File Path

The bundle command SHALL support overriding the default bundle file location.

#### Scenario: File flag overrides default

- **WHEN** user runs `cly bundle <type> --file /path/to/file`
- **THEN** command SHALL read packages from specified path
- **AND** ignore default bundle file location

### Requirement: Bundle File Format

Bundle files SHALL use a simple line-based format with comment support.

#### Scenario: Comments ignored

- **WHEN** bundle file contains lines starting with #
- **THEN** those lines SHALL be ignored during parsing

#### Scenario: Empty lines ignored

- **WHEN** bundle file contains empty lines or whitespace-only lines
- **THEN** those lines SHALL be ignored during parsing

#### Scenario: One package per line

- **WHEN** bundle file is parsed
- **THEN** each non-comment non-empty line SHALL be treated as one package

### Requirement: State Tracking

The bundle command SHALL track installed packages to detect removals.

#### Scenario: State file updated after sync

- **WHEN** sync completes successfully
- **THEN** state file SHALL contain list of successfully installed packages
- **AND** failed installs SHALL not be saved to state

#### Scenario: Removed packages detected

- **WHEN** package exists in state file but not in bundle file
- **THEN** package SHALL be marked for removal
- **AND** uninstall command SHALL execute for that package

### Requirement: Dependency Verification

The bundle command SHALL verify required tools exist before execution.

#### Scenario: Missing brew

- **WHEN** user runs `cly bundle brew` without brew installed
- **THEN** command SHALL error with message to install Homebrew

#### Scenario: Missing bun

- **WHEN** user runs `cly bundle js` without bun installed
- **THEN** command SHALL error with message to install bun

#### Scenario: Missing go

- **WHEN** user runs `cly bundle go` without go installed
- **THEN** command SHALL error with message to install go

#### Scenario: Missing uv

- **WHEN** user runs `cly bundle python` without uv installed
- **THEN** command SHALL error with message to install uv

### Requirement: Go Bundler Specifics

The go bundler SHALL support mise integration for GOPATH management.

#### Scenario: Mise detected

- **WHEN** mise is available and manages go
- **THEN** GOBIN SHALL be set to mise go bin directory
- **AND** packages install to mise-managed location

#### Scenario: Mise not available

- **WHEN** mise is not available
- **THEN** GOBIN SHALL fallback to `go env GOPATH`/bin

### Requirement: JS Bundler Specifics

The js bundler SHALL normalize GitHub package references.

#### Scenario: GitHub shorthand converted

- **WHEN** package is in format `user/repo` (not starting with @)
- **THEN** bun install SHALL receive `github:user/repo`

#### Scenario: Scoped packages preserved

- **WHEN** package is in format `@scope/package`
- **THEN** bun install SHALL receive package unchanged

### Requirement: Error Reporting

The bundle command SHALL report failures clearly while continuing with remaining packages.

#### Scenario: Install failure continues

- **WHEN** package install fails
- **THEN** command SHALL continue with remaining packages
- **AND** report failed packages at end

#### Scenario: Exit code reflects failures

- **WHEN** any package install fails
- **THEN** command SHALL exit with non-zero code
- **AND** successful packages SHALL still be recorded in state
