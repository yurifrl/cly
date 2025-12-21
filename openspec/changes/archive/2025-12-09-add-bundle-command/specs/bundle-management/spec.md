# bundle-management Specification

## Purpose

Declarative package management for multiple ecosystems (brew, go, js, python) through a unified CLI interface.

## ADDED Requirements

### Requirement: Bundle Command Structure

The bundle command SHALL provide subcommands for sync, check, and cleanup operations.

#### Scenario: Default sync operation

- **WHEN** user runs `cly bundle [type]`
- **THEN** command SHALL install packages from bundle file
- **AND** remove packages not in bundle file (cleanup)
- **AND** default type SHALL be brew

#### Scenario: Check shows diff

- **WHEN** user runs `cly bundle check [type]`
- **THEN** command SHALL show packages to install
- **AND** show packages to remove
- **AND** make no changes
- **AND** exit 0 if in sync, exit 1 if changes needed

#### Scenario: Cleanup removes only

- **WHEN** user runs `cly bundle cleanup [type]`
- **THEN** command SHALL remove packages not in bundle file
- **AND** not install any packages

#### Scenario: Help shows subcommands

- **WHEN** user runs `cly bundle --help`
- **THEN** output SHALL list check and cleanup subcommands
- **AND** list available bundle types

### Requirement: Flags

The bundle command SHALL support file override and verbose output.

#### Scenario: File flag overrides default

- **WHEN** user runs `cly bundle [type] --file /path/to/file`
- **THEN** command SHALL read from specified path
- **AND** ignore default bundle file location

#### Scenario: Verbose flag shows details

- **WHEN** user runs `cly bundle [type] --verbose`
- **THEN** command SHALL show detailed install/uninstall output

### Requirement: State Tracking via Store

Go, js, and python bundlers SHALL track state via injected Store.

#### Scenario: Install adds to store

- **WHEN** package install succeeds
- **THEN** package SHALL be added to Store with bundler type as namespace

#### Scenario: Cleanup removes from store

- **WHEN** package is uninstalled during cleanup
- **THEN** package SHALL be removed from Store

#### Scenario: Store diff determines changes

- **WHEN** sync or cleanup runs
- **THEN** to_remove = Store.List(type) - bundle_file_packages

### Requirement: Brew Delegation

Brew bundler SHALL delegate to brew bundle command.

#### Scenario: Brew sync

- **WHEN** user runs `cly bundle brew`
- **THEN** command SHALL execute `brew bundle --file=~/.config/Brewfile`

#### Scenario: Brew check

- **WHEN** user runs `cly bundle check brew`
- **THEN** command SHALL execute `brew bundle check --file=~/.config/Brewfile`

#### Scenario: Brew cleanup

- **WHEN** user runs `cly bundle cleanup brew`
- **THEN** command SHALL execute `brew bundle cleanup --file=~/.config/Brewfile --force`

### Requirement: Bundle File Format

Bundle files SHALL use simple line-based format.

#### Scenario: Comments ignored

- **WHEN** bundle file contains lines starting with #
- **THEN** those lines SHALL be ignored

#### Scenario: Empty lines ignored

- **WHEN** bundle file contains empty or whitespace-only lines
- **THEN** those lines SHALL be ignored

### Requirement: Dependency Verification

The bundle command SHALL verify required tools exist.

#### Scenario: Missing tool

- **WHEN** required tool is not installed
- **THEN** command SHALL error with install instructions

### Requirement: Go Bundler Specifics

The go bundler SHALL support mise integration and binary removal.

#### Scenario: Mise detection

- **WHEN** mise manages go
- **THEN** GOBIN SHALL use mise go bin directory

#### Scenario: Binary removal

- **WHEN** go package is cleaned up
- **THEN** binary SHALL be removed from GOBIN

### Requirement: JS Bundler Specifics

The js bundler SHALL normalize GitHub shorthand and preserve scoped packages.

#### Scenario: GitHub shorthand

- **WHEN** package is `user/repo` (not @-prefixed)
- **THEN** bun install SHALL receive `github:user/repo`

#### Scenario: Scoped packages

- **WHEN** package is `@scope/pkg`
- **THEN** bun install SHALL receive unchanged

### Requirement: Self-Healing State

The bundle command SHALL self-heal when Store is out of sync with reality.

#### Scenario: Out of sync cleanup

- **WHEN** Store says package installed but uninstall fails
- **THEN** log warning and remove from Store anyway

#### Scenario: Manual install untouched

- **WHEN** package is installed but not in Store
- **THEN** cleanup SHALL not touch it
