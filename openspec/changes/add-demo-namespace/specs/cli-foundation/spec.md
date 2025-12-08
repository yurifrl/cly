# cli-foundation Specification Delta

## ADDED Requirements

### Requirement: Command Namespaces
The application SHALL support hierarchical command namespaces for organizing related subcommands.

#### Scenario: Parent command with subcommands
- **WHEN** a namespace is created (e.g., `demo`)
- **THEN** it SHALL be a parent Cobra command with no RunE function
- **AND** subcommands SHALL register with the parent command
- **AND** parent command SHALL register with RootCmd
- **AND** usage SHALL show `cly <namespace> <subcommand>` format

#### Scenario: Namespace directory structure
- **WHEN** implementing a namespace
- **THEN** namespace SHALL have directory `modules/<namespace>/cmd.go`
- **AND** subcommands SHALL live in `modules/<namespace>/<subcommand>/`
- **AND** each subcommand SHALL have `cmd.go` and implementation file
