# demo-commands Specification

## Purpose
TBD - created by archiving change add-demo-namespace. Update Purpose after archive.
## Requirements
### Requirement: Demo Namespace
The application SHALL provide a `demo` namespace for showcasing Charm components.

#### Scenario: Demo parent command exists
- **WHEN** user runs `cly --help`
- **THEN** output SHALL include `demo` command in available commands
- **AND** show description about component demonstrations

#### Scenario: Demo command shows subcommands
- **WHEN** user runs `cly demo --help`
- **THEN** output SHALL list available demo subcommands
- **AND** each subcommand SHALL have description of component it demonstrates

### Requirement: Chat Demo Subcommand
The application SHALL provide a chat demo accessible via `demo chat` that demonstrates textarea and viewport components.

#### Scenario: Chat command is accessible
- **WHEN** user runs `cly demo --help`
- **THEN** `chat` SHALL appear in subcommands list
- **AND** show description "Interactive chat demo (textarea + viewport)"

#### Scenario: Chat demo launches interactive UI
- **WHEN** user runs `cly demo chat`
- **THEN** an interactive chat interface SHALL be displayed
- **AND** viewport SHALL show welcome message
- **AND** textarea SHALL be focused and ready for input
- **AND** textarea placeholder SHALL show "Send a message..."

### Requirement: Chat Message Handling
The chat demo SHALL accept user input and display messages in scrollable history.

#### Scenario: User sends message
- **WHEN** user types text in textarea and presses Enter
- **THEN** message SHALL be prefixed with "You: " in colored style
- **AND** message SHALL be appended to viewport content
- **AND** viewport SHALL scroll to bottom automatically
- **AND** textarea SHALL be cleared for next message

#### Scenario: Multiple messages scroll
- **WHEN** multiple messages fill viewport height
- **THEN** viewport SHALL become scrollable
- **AND** older messages SHALL scroll up
- **AND** newest message SHALL remain visible at bottom

### Requirement: Chat Demo Controls
The chat demo SHALL support standard keyboard navigation and exit.

#### Scenario: User exits with Esc
- **WHEN** user presses Esc key
- **THEN** program SHALL print current textarea value to stdout
- **AND** exit gracefully with code 0

#### Scenario: User exits with Ctrl+C
- **WHEN** user presses Ctrl+C
- **THEN** program SHALL print current textarea value to stdout
- **AND** exit gracefully with code 0

### Requirement: Chat Demo Layout
The chat demo SHALL handle terminal resizing and maintain proper layout.

#### Scenario: Window resize adjusts layout
- **WHEN** terminal window is resized
- **THEN** viewport width SHALL match new window width
- **AND** textarea width SHALL match new window width
- **AND** viewport height SHALL adjust to fill available space minus textarea
- **AND** message content SHALL be re-wrapped to new width
- **AND** viewport SHALL remain scrolled to bottom

### Requirement: Namespace Pattern
The demo namespace SHALL follow a hierarchical command structure for subcommands.

#### Scenario: Subcommand registration pattern
- **WHEN** demo namespace module is implemented
- **THEN** it SHALL have parent command in `modules/demo/cmd.go`
- **AND** each subcommand SHALL live in `modules/demo/<name>/`
- **AND** subcommands SHALL register with demo parent in demo module init()
- **AND** demo parent SHALL register with RootCmd in root init()

#### Scenario: Command hierarchy in help
- **WHEN** user runs `cly demo chat --help`
- **THEN** usage SHALL show `cly demo chat [flags]`
- **AND** demonstrate proper command nesting

