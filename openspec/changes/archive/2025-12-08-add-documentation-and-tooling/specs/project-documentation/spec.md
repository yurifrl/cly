# Project Documentation Specification

## ADDED Requirements

### Requirement: README File
The project SHALL provide a README.md file at the repository root with comprehensive project information.

#### Scenario: README exists and is discoverable
- **WHEN** user views the repository on GitHub or locally
- **THEN** README.md SHALL exist at project root
- **AND** be automatically displayed on GitHub repository page

#### Scenario: README contains project overview
- **WHEN** user reads README
- **THEN** it SHALL include project name "CLY - Command Line Utilities"
- **AND** include tagline describing it as modular CLI with Charm libraries
- **AND** include purpose statement about day-to-day utilities with beautiful TUIs

### Requirement: Installation Instructions
The README SHALL provide clear installation instructions for multiple methods.

#### Scenario: Go install method documented
- **WHEN** user wants to install via go install
- **THEN** README SHALL include `go install github.com/yurifrl/cly@latest` command
- **AND** specify it installs to $GOPATH/bin

#### Scenario: Build from source method documented
- **WHEN** user wants to build from source
- **THEN** README SHALL include git clone command
- **AND** include `go build` instructions
- **AND** specify resulting binary name is `cly`

### Requirement: Usage Examples
The README SHALL provide concrete usage examples for all command types.

#### Scenario: UUID utility examples shown
- **WHEN** user wants to learn UUID command
- **THEN** README SHALL show `cly uuid` command
- **AND** explain interactive selection interface

#### Scenario: Demo command examples shown
- **WHEN** user wants to explore demos
- **THEN** README SHALL show `cly demo --help` command
- **AND** list examples like `cly demo chat`, `cly demo spinner`
- **AND** mention 48 Bubbletea examples available

### Requirement: Commands Reference
The README SHALL provide a complete reference of available commands.

#### Scenario: Commands table exists
- **WHEN** user needs command overview
- **THEN** README SHALL include table with Command and Description columns
- **AND** table SHALL list uuid utility
- **AND** table SHALL list demo namespace with count of examples

#### Scenario: Command categories are clear
- **WHEN** viewing commands table
- **THEN** utilities (uuid) SHALL be distinguished from demos
- **AND** demo namespace SHALL indicate it contains multiple subcommands

### Requirement: Architecture Documentation
The README SHALL explain the project's modular architecture.

#### Scenario: Architecture section exists
- **WHEN** user wants to understand structure
- **THEN** README SHALL include architecture section
- **AND** explain modular design with zero coupling
- **AND** show directory structure for modules/

#### Scenario: Module addition process documented
- **WHEN** developer wants to add new module
- **THEN** README SHALL reference docs/module-template.md
- **AND** describe high-level steps (copy, adapt, register)

### Requirement: Dependencies Documentation
The README SHALL list all external library dependencies.

#### Scenario: Core dependencies listed
- **WHEN** user needs dependency information
- **THEN** README SHALL list Cobra with GitHub link
- **AND** list Bubbletea with GitHub link
- **AND** list Bubbles with GitHub link
- **AND** list Lipgloss with GitHub link
- **AND** describe purpose of each dependency
