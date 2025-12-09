# session-management Specification

## Purpose
Provides named CLI session management with terminal integration and environment variable exports for better user experience and context tracking.

## ADDED Requirements

### Requirement: Session Name Sources
The system SHALL support multiple sources for session names with defined precedence.

#### Scenario: Explicit flag with value
- **WHEN** user runs `cli --name WorkProject`
- **THEN** session name SHALL be set to `WorkProject`
- **AND** print `🏷️  Session: WorkProject`

#### Scenario: Explicit flag without value triggers auto-generation
- **WHEN** user runs `cli --name` without a value
- **THEN** session name SHALL be auto-generated in two-word format
- **AND** print `🏷️  Session: [GeneratedName]`

#### Scenario: No flag defaults to auto-generation
- **WHEN** user runs `cli` without `--name` flag
- **THEN** session name SHALL be auto-generated in two-word format
- **AND** print `🏷️  Session: [GeneratedName]`

#### Scenario: Environment variable precedence
- **WHEN** `CLY_SESSION_NAME` environment variable is set
- **AND** user does not provide `--name` flag
- **THEN** session name SHALL use the environment variable value
- **AND** print `🏷️  Session: [EnvValue]`

### Requirement: Auto-Generated Names
The system SHALL generate memorable two-word session names when explicit names are not provided.

#### Scenario: Name format follows pattern
- **WHEN** session name is auto-generated
- **THEN** format SHALL be `[TitleCaseWord1][TitleCaseWord2]`
- **AND** examples include `QuickTask`, `TempWork`, `BrightIdea`

#### Scenario: Word pools provide variety
- **WHEN** generating names
- **THEN** system SHALL use word pools for colors, animals, adjectives, nouns
- **AND** combinations SHALL be random
- **AND** names SHALL be memorable and human-readable

### Requirement: Environment Variable Export
The system SHALL export session information as environment variables for downstream tools.

#### Scenario: Export session name
- **WHEN** session is initialized with name `WorkProject`
- **THEN** `CLY_SESSION_NAME` environment variable SHALL be exported with value `WorkProject`
- **AND** variable SHALL be available to child processes

### Requirement: Terminal Integration - Zellij
The system SHALL integrate with Zellij terminal to update tab and pane names.

#### Scenario: Detect Zellij environment
- **WHEN** session starts in Zellij terminal
- **THEN** system SHALL detect Zellij by checking environment variables
- **AND** proceed with terminal integration

#### Scenario: Update tab name
- **WHEN** session is initialized in Zellij
- **THEN** tab name SHALL be updated to match session name
- **AND** use Zellij escape sequences for updates

#### Scenario: Update pane name
- **WHEN** session is initialized in Zellij
- **THEN** pane name SHALL be updated to match session name
- **AND** use Zellij escape sequences for updates

#### Scenario: Skip integration in non-Zellij terminals
- **WHEN** session starts in non-Zellij terminal
- **THEN** terminal integration SHALL be skipped
- **AND** session SHALL function normally without errors

### Requirement: Session Initialization
The system SHALL initialize session state before command execution.

#### Scenario: Initialize in root command PreRun
- **WHEN** any CLI command is executed
- **THEN** session SHALL be initialized in `PersistentPreRunE`
- **AND** initialization SHALL complete before command execution
- **AND** errors SHALL be returned to halt execution if initialization fails

#### Scenario: Print session indicator
- **WHEN** session is initialized successfully
- **THEN** output SHALL display `🏷️  Session: [SessionName]`
- **AND** output SHALL appear before command execution

### Requirement: Session Name Validation
The system SHALL validate session names to ensure they are safe and readable.

#### Scenario: Accept valid characters
- **WHEN** session name contains alphanumeric characters
- **OR** session name contains hyphens or underscores
- **THEN** validation SHALL pass

#### Scenario: Reject invalid characters
- **WHEN** session name contains special characters other than hyphen and underscore
- **OR** session name contains spaces
- **THEN** validation SHALL fail with clear error message
