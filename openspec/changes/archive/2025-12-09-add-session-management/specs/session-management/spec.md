# session-management Specification

## Purpose
Provides named CLI session management with terminal integration and environment variable exports for the `cly claude` command.

## ADDED Requirements

### Requirement: Claude Command
The system SHALL provide a `cly claude` command that wraps Claude Code with session management.

#### Scenario: Run claude with session
- **WHEN** user runs `cly claude`
- **THEN** session SHALL be initialized
- **AND** Claude Code SHALL be executed with session context

#### Scenario: Pass arguments to claude
- **WHEN** user runs `cly claude [args...]`
- **THEN** all arguments SHALL be passed through to Claude Code

### Requirement: Session Name Sources
The system SHALL support multiple sources for session names with defined precedence.

#### Scenario: Explicit flag with value
- **WHEN** user runs `cly claude --name WorkProject`
- **THEN** session name SHALL be set to `WorkProject`
- **AND** print `🏷️  Session: WorkProject`

#### Scenario: Explicit flag without value triggers auto-generation
- **WHEN** user runs `cly claude --name` without a value
- **THEN** session name SHALL be auto-generated in two-word format
- **AND** print `🏷️  Session: [GeneratedName]`

#### Scenario: No flag defaults to auto-generation
- **WHEN** user runs `cly claude` without `--name` flag
- **THEN** session name SHALL be auto-generated in two-word format
- **AND** print `🏷️  Session: [GeneratedName]`

#### Scenario: Environment variable precedence
- **WHEN** `CLAUDE_SESSION_NAME` environment variable is set
- **AND** user does not provide `--name` flag
- **THEN** session name SHALL use the environment variable value
- **AND** print `🏷️  Session: [EnvValue]`

### Requirement: Auto-Generated Names
The system SHALL generate memorable two-word session names when explicit names are not provided.

#### Scenario: Name format follows pattern
- **WHEN** session name is auto-generated
- **THEN** format SHALL be `[TitleCaseWord1][TitleCaseWord2]`
- **AND** examples include `QuickFox`, `BrightOwl`, `SwiftBear`

#### Scenario: Word pools provide variety
- **WHEN** generating names
- **THEN** system SHALL use word pools (adjectives + animals)
- **AND** combinations SHALL be random
- **AND** names SHALL be memorable and human-readable

### Requirement: Environment Variable Export
The system SHALL export session information as environment variables for downstream tools.

#### Scenario: Export session name
- **WHEN** session is initialized with name `WorkProject`
- **THEN** `CLAUDE_SESSION_NAME` environment variable SHALL be exported with value `WorkProject`
- **AND** variable SHALL be available to Claude Code process

### Requirement: Terminal Integration - Zellij
The system SHALL integrate with Zellij terminal to update tab names.

#### Scenario: Detect Zellij environment
- **WHEN** session starts in Zellij terminal
- **THEN** system SHALL detect Zellij by checking `$ZELLIJ` environment variable
- **AND** proceed with terminal integration

#### Scenario: Update tab name
- **WHEN** session is initialized in Zellij
- **THEN** tab name SHALL be updated to match session name
- **AND** use `zellij action rename-tab` command

#### Scenario: Skip integration in non-Zellij terminals
- **WHEN** session starts in non-Zellij terminal
- **THEN** terminal integration SHALL be skipped silently
- **AND** session SHALL function normally

### Requirement: Session Name Validation
The system SHALL validate session names to ensure they are safe and readable.

#### Scenario: Accept valid characters
- **WHEN** session name contains alphanumeric characters, hyphens, or underscores
- **THEN** validation SHALL pass

#### Scenario: Reject invalid characters
- **WHEN** session name contains special characters other than hyphen and underscore
- **OR** session name contains spaces
- **THEN** validation SHALL fail with clear error message
