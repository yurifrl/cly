# dotfiles-management Specification Delta

## MODIFIED Requirements

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
