# cli-foundation Specification

## Purpose
TBD - created by archiving change scaffold-foundation. Update Purpose after archive.
## Requirements
### Requirement: Go Module Initialization
The project SHALL be initialized as a Go module with proper naming and dependency management.

#### Scenario: Module created with correct name
- **WHEN** `go mod init` is executed
- **THEN** `go.mod` SHALL exist with module name `github.com/yurifrl/cly`
- **AND** Go version SHALL be specified as `1.24` or compatible

#### Scenario: Core dependencies installed
- **WHEN** dependencies are installed
- **THEN** `go.mod` SHALL include `github.com/spf13/cobra` for CLI framework
- **AND** `go.mod` SHALL include `github.com/charmbracelet/lipgloss` for styling
- **AND** `go mod tidy` SHALL execute without errors

### Requirement: Entry Point
The application SHALL have a minimal main entry point that delegates to the command framework.

#### Scenario: Main function executes root command
- **WHEN** application starts
- **THEN** `main.go` SHALL call `cmd.Execute()`
- **AND** exit with code 1 if Execute returns an error
- **AND** exit with code 0 on success

#### Scenario: Executable can be built
- **WHEN** `go build` is run
- **THEN** compilation SHALL succeed without errors
- **AND** produce a single binary executable named `cly`

### Requirement: Root Command
The application SHALL provide a root command with styled help output and proper CLI structure.

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

### Requirement: Shared Styling
The application SHALL provide consistent theming through shared Lipgloss styles.

#### Scenario: Theme package exports standard styles
- **WHEN** `pkg/style/theme.go` is imported
- **THEN** it SHALL export `TitleStyle` with bold and foreground color "212" (purple/pink)
- **AND** export `SubtleStyle` with foreground color "241" (gray)
- **AND** styles SHALL be Lipgloss.Style instances ready for immediate use

#### Scenario: Styles are reusable across modules
- **WHEN** any module imports `cly/pkg/style`
- **THEN** it SHALL access all exported styles without redefinition
- **AND** styles SHALL be consistent across all command outputs

### Requirement: Project Structure
The project SHALL establish a modular directory structure for scalable command addition.

#### Scenario: Directory organization follows specification
- **WHEN** foundation is scaffolded
- **THEN** `cmd/` directory SHALL exist for command definitions
- **AND** `pkg/` directory SHALL exist for shared utilities
- **AND** `pkg/style/` subdirectory SHALL exist for styling
- **AND** `modules/` directory SHALL exist as placeholder for future commands

#### Scenario: Files match Phase 1 specification
- **WHEN** foundation implementation is complete
- **THEN** file structure SHALL match:
  - `main.go` at project root
  - `go.mod` and `go.sum` at project root
  - `cmd/root.go` for root command
  - `pkg/style/theme.go` for shared styles

### Requirement: Module Registration
The application SHALL support modular command registration through a standardized pattern.

#### Scenario: Module registration in init function
- **WHEN** a module is added to the application
- **THEN** it SHALL be registered in `cmd/root.go` init() function
- **AND** registration SHALL call module's Register(RootCmd) function
- **AND** init() SHALL execute before main()

#### Scenario: Multiple modules can be registered
- **WHEN** multiple modules are registered
- **THEN** each module SHALL register independently in init()
- **AND** modules SHALL appear in help output
- **AND** adding/removing modules SHALL only require changes to cmd/root.go init()

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

