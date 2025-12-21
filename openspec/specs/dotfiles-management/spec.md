# dotfiles-management Specification

## Purpose
TBD - created by archiving change add-dotfiles-command. Update Purpose after archive.
## Requirements
### Requirement: Dotfiles Command

The application SHALL provide a dotfiles command for symlink management.

#### Scenario: Basic sync

- **WHEN** user runs `cly dotfiles`
- **THEN** config file SHALL be read from configured directory
- **AND** symlinks SHALL be created for each mapping
- **AND** install commands SHALL be skipped

#### Scenario: Install mode

- **WHEN** user runs `cly dotfiles -i`
- **THEN** symlinks SHALL be created
- **AND** install commands (`!` prefixed) SHALL be executed

### Requirement: Config File Format

The application SHALL parse dotfiles.conf with specific format.

#### Scenario: Mapping syntax

- **WHEN** config contains `./source -> ~/destination`
- **THEN** source path SHALL be resolved relative to config directory
- **AND** destination path SHALL expand `~` to home directory
- **AND** symlink SHALL be created from destination to source

#### Scenario: Directory mapping

- **WHEN** config contains `./source/ -> ~/destination/`
- **THEN** trailing slash SHALL indicate directory
- **AND** source SHALL be validated as directory
- **AND** symlink SHALL link entire directory

#### Scenario: Comments and empty lines

- **WHEN** config contains lines starting with `#` or empty lines
- **THEN** those lines SHALL be skipped
- **AND** processing SHALL continue with next line

#### Scenario: Invalid format

- **WHEN** config contains line without `->` separator
- **THEN** error SHALL be reported with line number
- **AND** processing SHALL continue with warning

### Requirement: Symlink Operations

The application SHALL handle symlink creation and conflicts.

#### Scenario: Create new symlink

- **WHEN** destination does not exist
- **THEN** parent directories SHALL be created if needed
- **AND** symlink SHALL be created pointing to source

#### Scenario: Replace existing symlink

- **WHEN** destination is an existing symlink
- **THEN** existing symlink SHALL be removed
- **AND** new symlink SHALL be created

#### Scenario: Conflict with existing file

- **WHEN** destination is a regular file or directory (not symlink)
- **THEN** error SHALL be reported
- **AND** symlink SHALL NOT be created
- **AND** processing SHALL continue with next mapping

#### Scenario: Missing source

- **WHEN** source path does not exist
- **THEN** warning SHALL be displayed
- **AND** mapping SHALL be skipped
- **AND** processing SHALL continue

#### Scenario: Directory type mismatch

- **WHEN** source is directory but config line has no trailing slash
- **THEN** error SHALL be reported
- **AND** hint SHALL suggest adding trailing slash

### Requirement: Install Commands

The application SHALL support shell command execution.

#### Scenario: Zellij plugin download

- **WHEN** install command is `zellij_plugin https://github.com/owner/repo`
- **THEN** owner and repo SHALL be extracted from URL
- **AND** file SHALL be downloaded from `https://github.com/owner/repo/releases/latest/download/repo.wasm`
- **AND** file SHALL be saved to `<zellij_plugins_dir>/repo.wasm`
- **AND** plugins directory SHALL be created if it does not exist
- **AND** `<zellij_plugins_dir>` SHALL be configurable via `modules.dotfiles.zellij_plugins_dir`
- **AND** SHALL default to `~/.config/zellij/plugins` if not configured

#### Scenario: Invalid GitHub URL

- **WHEN** install command is `zellij_plugin https://invalid.com/foo`
- **THEN** error SHALL be displayed
- **AND** processing SHALL continue with next command

#### Scenario: Download failure

- **WHEN** release asset does not exist or network fails
- **THEN** error SHALL be displayed with URL attempted
- **AND** processing SHALL continue with next command

### Requirement: Config Resolution

The application SHALL resolve config file location.

#### Scenario: Default directory

- **WHEN** no `--config` flag provided
- **THEN** directory SHALL be read from `modules.dotfiles.directory` config
- **AND** config file SHALL be `<directory>/dotfiles.conf`

#### Scenario: Config flag override

- **WHEN** user provides `--config /path/to/dotfiles.conf`
- **THEN** specified path SHALL be used
- **AND** relative paths in config SHALL resolve to config file's directory

#### Scenario: Missing config

- **WHEN** config file does not exist
- **THEN** error SHALL be displayed with expected path
- **AND** hint SHALL suggest creating config or using `--config` flag

### Requirement: Status Subcommand

The application SHALL provide status display.

#### Scenario: Status output

- **WHEN** user runs `cly dotfiles status`
- **THEN** each mapping SHALL be displayed with state
- **AND** states SHALL be: linked, missing, conflict, broken
- **AND** install command count SHALL be shown

#### Scenario: Status states

- **WHEN** checking mapping state
- **THEN** "linked" SHALL indicate symlink exists and points to source
- **AND** "missing" SHALL indicate source does not exist
- **AND** "conflict" SHALL indicate destination is regular file/directory
- **AND** "broken" SHALL indicate symlink exists but target missing

### Requirement: Unlink Subcommand

The application SHALL provide symlink removal.

#### Scenario: Unlink all

- **WHEN** user runs `cly dotfiles unlink`
- **THEN** all symlinks from config mappings SHALL be removed
- **AND** regular files/directories SHALL be skipped
- **AND** count of removed symlinks SHALL be displayed

#### Scenario: Unlink non-existent

- **WHEN** destination does not exist
- **THEN** mapping SHALL be skipped silently
- **AND** processing SHALL continue

