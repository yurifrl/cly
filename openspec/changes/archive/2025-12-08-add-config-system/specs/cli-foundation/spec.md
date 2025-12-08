# cli-foundation Specification Delta

## MODIFIED Requirements

### Requirement: Root Command
The application SHALL provide a root command with styled help output, proper CLI structure, and configuration initialization.

#### Scenario: Help flag displays styled information
- **WHEN** user runs `cly --help` or `cly -h`
- **THEN** output SHALL display application name styled with Lipgloss TitleStyle
- **AND** include short description "Charm Libraries Showcase"
- **AND** include long description with component list (spinner, textinput, list, table)
- **AND** include usage instructions with "Press 'q' or Ctrl+C to quit"

#### Scenario: Root command structure supports module registration
- **WHEN** root command is defined in `cmd/root.go`
- **THEN** it SHALL export a `RootCmd` variable of type `*cobra.Command`
- **AND** provide an `Execute()` function that returns error
- **AND** use Cobra's standard command structure for future module registration

#### Scenario: Configuration loads before command execution
- **WHEN** any command is executed
- **THEN** root command SHALL have PersistentPreRunE function
- **AND** PersistentPreRunE SHALL call config.Load()
- **AND** config SHALL be loaded before subcommand RunE executes
- **AND** config errors SHALL prevent command execution
