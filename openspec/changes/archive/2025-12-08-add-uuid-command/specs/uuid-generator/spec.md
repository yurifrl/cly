# UUID Generator Specification

## ADDED Requirements

### Requirement: UUID Generation Capability
The application SHALL provide a UUID generation utility accessible via the `uuid` command.

#### Scenario: Command is registered and accessible
- **WHEN** user runs `cly --help`
- **THEN** output SHALL include `uuid` command in the available commands list
- **AND** show description "Generate UUIDs"

#### Scenario: Command launches interactive TUI
- **WHEN** user runs `cly uuid`
- **THEN** an interactive list SHALL be displayed
- **AND** list SHALL show title "Generate UUID"
- **AND** list SHALL contain 3 options: "UUID v4 (random)", "UUID v7 (time-ordered)", "Multiple (5x)"

### Requirement: UUID v4 Generation
The application SHALL generate UUID version 4 (random) when selected.

#### Scenario: v4 UUID generated successfully
- **WHEN** user selects "UUID v4 (random)" option and presses Enter
- **THEN** a valid UUID v4 SHALL be generated
- **AND** UUID SHALL be displayed in green color (color 42)
- **AND** UUID SHALL be printed to stdout
- **AND** program SHALL exit after displaying result

#### Scenario: v4 UUID format validation
- **WHEN** UUID v4 is generated
- **THEN** it SHALL match format: `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx`
- **AND** version field SHALL be `4`
- **AND** variant SHALL be RFC 4122 compliant

### Requirement: UUID v7 Generation
The application SHALL generate UUID version 7 (time-ordered) when selected.

#### Scenario: v7 UUID generated successfully
- **WHEN** user selects "UUID v7 (time-ordered)" option and presses Enter
- **THEN** a valid UUID v7 SHALL be generated
- **AND** UUID SHALL be displayed in green color (color 42)
- **AND** UUID SHALL be printed to stdout
- **AND** program SHALL exit after displaying result

#### Scenario: v7 UUID is time-ordered
- **WHEN** multiple UUID v7s are generated in sequence
- **THEN** each subsequent UUID SHALL be lexicographically sortable by time
- **AND** version field SHALL be `7`

### Requirement: Multiple UUID Generation
The application SHALL generate multiple UUIDs (5) when selected.

#### Scenario: Multiple UUIDs generated
- **WHEN** user selects "Multiple (5x)" option and presses Enter
- **THEN** exactly 5 UUID v4s SHALL be generated
- **AND** each UUID SHALL be on a separate line
- **AND** all UUIDs SHALL be displayed in green color (color 42)
- **AND** program SHALL exit after displaying results

### Requirement: Interactive List Navigation
The application SHALL provide keyboard-based navigation for UUID type selection.

#### Scenario: Arrow key navigation
- **WHEN** list is displayed
- **THEN** user SHALL be able to navigate options using up/down arrow keys
- **AND** current selection SHALL be visually highlighted
- **AND** Enter key SHALL confirm selection and generate UUID

#### Scenario: Cancellation without generation
- **WHEN** user presses 'q' or 'Ctrl+C' before selecting an option
- **THEN** program SHALL display "Cancelled."
- **AND** no UUID SHALL be generated
- **AND** program SHALL exit with code 0

### Requirement: Module Registration Pattern
The uuid module SHALL follow the established modular registration pattern.

#### Scenario: Module self-registration
- **WHEN** uuid package is imported in `cmd/root.go`
- **THEN** it SHALL expose a `Register(parent *cobra.Command)` function
- **AND** Register function SHALL add the uuid command to the parent
- **AND** registration SHALL occur in `init()` function of root command

#### Scenario: Module isolation
- **WHEN** uuid module is implemented
- **THEN** it SHALL be self-contained in `modules/uuid/` directory
- **AND** SHALL NOT import any other modules (only pkg/ and external deps)
- **AND** SHALL have separate files: `cmd.go` (registration) and `uuid.go` (implementation)
