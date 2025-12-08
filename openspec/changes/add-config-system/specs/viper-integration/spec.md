# Viper Integration Specification

## ADDED Requirements

### Requirement: Viper Configuration Loader
The application SHALL use Viper library for configuration management.

#### Scenario: Viper initialization
- **WHEN** application starts
- **THEN** Viper SHALL be configured with config name "config"
- **AND** add search paths: ~/.config/cly, . (current directory)
- **AND** set env prefix to "CLY"
- **AND** enable automatic env var binding

#### Scenario: Config file discovery
- **WHEN** Viper searches for config
- **THEN** it SHALL add search paths in order: ~/.config/cly, . (current dir)
- **AND** Viper SHALL check paths in order added
- **AND** use first config.yaml found
- **AND** fall back to embedded config/config.yaml if none found
- **AND** not error if no external config file exists

### Requirement: Config Struct Unmarshaling
The configuration SHALL be unmarshaled into a Go struct for type safety.

#### Scenario: Config struct definition
- **WHEN** config is loaded
- **THEN** it SHALL unmarshal into Config struct
- **AND** struct SHALL have App, Theme, Modules fields
- **AND** provide type-safe access to values
- **AND** support nested structs for modules

#### Scenario: Config validation
- **WHEN** config is unmarshaled
- **THEN** invalid YAML SHALL return error
- **AND** unknown fields SHALL be ignored (forward compatibility)
- **AND** required fields SHALL have default values
- **AND** type mismatches SHALL return error

### Requirement: Hot Reload Support
The configuration SHALL support runtime reloading without restart.

#### Scenario: Config file changes detected
- **WHEN** user modifies ~/.config/cly/config.yaml
- **THEN** application MAY detect changes via Viper.WatchConfig()
- **AND** reload configuration automatically
- **AND** log reload event if debug enabled

#### Scenario: Reload without breaking sessions
- **WHEN** config reloads during active TUI
- **THEN** running demos SHALL not be interrupted
- **AND** new config SHALL apply to next command execution
- **AND** active sessions SHALL use config from their start time
