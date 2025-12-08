# Config Management Specification

## ADDED Requirements

### Requirement: Default Configuration File
The application SHALL provide a default configuration file with sensible defaults.

#### Scenario: Default config file exists
- **WHEN** application is built
- **THEN** `config/config.yaml` SHALL exist in source tree
- **AND** be embedded in the compiled binary
- **AND** provide fallback when user config does not exist

#### Scenario: Default config structure
- **WHEN** reading default config
- **THEN** it SHALL include `app` section with name, debug, version fields
- **AND** include `theme` section with style field
- **AND** include `modules` section for module-specific settings
- **AND** use valid YAML format

### Requirement: User Configuration File
The application SHALL support user-specific configuration overrides.

#### Scenario: User config location
- **WHEN** user wants to customize settings
- **THEN** config file SHALL be located at `~/.config/cly/config.yaml`
- **AND** this location SHALL be standard across all platforms
- **AND** user config SHALL override default values

#### Scenario: User config creation
- **WHEN** user runs `cly config init`
- **THEN** config file SHALL be created at `~/.config/cly/config.yaml`
- **AND** file SHALL contain default values as starting point
- **AND** command SHALL not overwrite existing config without confirmation

### Requirement: Environment Variable Support
The application SHALL support environment variable overrides with CLY_ prefix.

#### Scenario: Environment variable override
- **WHEN** `CLY_APP_DEBUG=true` is set
- **THEN** it SHALL override `app.debug` from config file
- **AND** use automatic Viper env binding with CLY prefix
- **AND** support nested keys with underscores (CLY_MODULES_UUID_DEFAULT_VERSION)

#### Scenario: Precedence order
- **WHEN** same setting exists in multiple places
- **THEN** precedence SHALL be (highest to lowest):
  1. Environment variables (CLY_*)
  2. User config (~/.config/cly/config.yaml)
  3. Embedded defaults (config/config.yaml)
  4. Current directory (./config.yaml)
- **AND** first found value SHALL be used

### Requirement: Config Commands
The application SHALL provide subcommands for config management.

#### Scenario: Config init command
- **WHEN** user runs `cly config init`
- **THEN** user config file SHALL be created
- **AND** prompt for confirmation if file exists
- **AND** display success message with file location

#### Scenario: Config show command
- **WHEN** user runs `cly config show`
- **THEN** current merged config SHALL be displayed
- **AND** show effective values (after env var overrides)
- **AND** format as YAML for readability

#### Scenario: Config get command
- **WHEN** user runs `cly config get <key>`
- **THEN** specific value SHALL be retrieved
- **AND** support dot notation (app.name, theme.style)
- **AND** return error if key does not exist

#### Scenario: Config set command
- **WHEN** user runs `cly config set <key> <value>`
- **THEN** value SHALL be written to user config file
- **AND** create file if it does not exist
- **AND** preserve existing values
- **AND** validate YAML after write

### Requirement: Module-Specific Configuration
The application SHALL support module-specific configuration sections.

#### Scenario: Module config namespace
- **WHEN** modules need configuration
- **THEN** config SHALL have `modules.<module-name>` section
- **AND** each module SHALL read from its own namespace
- **AND** modules SHALL have default values if config missing

#### Scenario: UUID module configuration
- **WHEN** UUID module reads config
- **THEN** it SHALL check `modules.uuid.default_version` (v4 or v7)
- **AND** use configured version as default in interactive list
- **AND** fall back to v4 if not configured

### Requirement: Theme Configuration
The application SHALL support theme customization via config.

#### Scenario: Theme style setting
- **WHEN** config contains `theme.style` value
- **THEN** pkg/style SHALL load corresponding theme (charm, dracula, catppuccin)
- **AND** apply theme colors to all TitleStyle, SubtleStyle, etc.
- **AND** fall back to charm theme if style is invalid
