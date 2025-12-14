# MCP Manager

Capability for managing MCP (Model Context Protocol) servers across AI tools.

## ADDED Requirements

### Requirement: Dual Binary Build

The module SHALL be buildable as both a CLY subcommand and a standalone binary.

#### Scenario: Build as CLY subcommand
Given the module is registered in cmd/cly/main.go
When `go build ./cmd/cly` runs
Then `cly mcp` command is available

#### Scenario: Build as standalone binary
Given cmd/mcp/main.go uses module as root
When `go build ./cmd/mcp` runs
Then standalone `mcp` binary is produced

#### Scenario: Same functionality in both
Given both binaries are built
When running `cly mcp list` and `mcp list`
Then output is identical

### Requirement: MCP Catalog Loading

The module SHALL load MCP definitions from source files.

#### Scenario: Load YAML source files
Given source files exist at `~/.config/mcpcli/mcps/*.yaml`
When the catalog is loaded
Then all MCPs are parsed with name, command, args, tags

#### Scenario: Handle missing directory
Given `~/.config/mcpcli/mcps/` does not exist
When the catalog is loaded
Then an empty catalog is returned without error

### Requirement: AI Tool Adapters

The module SHALL read and write MCP configurations for different AI tools.

#### Scenario: Read Claude Code config
Given `~/.claude.json` exists with `mcpServers` key
When adapter reads config
Then installed MCPs are returned as map of names

#### Scenario: Write Claude Code config
Given a list of MCPs to install
When adapter writes config
Then `~/.claude.json` is updated with `mcpServers` entries

#### Scenario: Read Cursor config
Given `~/.cursor/mcp.json` exists
When adapter reads config
Then installed MCPs are returned

### Requirement: Context Detection

The module SHALL determine which AI tool and scope to target.

#### Scenario: Explicit flag overrides all
Given `--context cursor:project` flag
When context is detected
Then cursor:project is used regardless of other sources

#### Scenario: Config file defaults
Given config has `defaults.ai: claude` and `defaults.scope: user`
When no flag is provided
Then claude:user is used

#### Scenario: Fallback to claude:user
Given no config and no flag
When context is detected
Then claude:user is the default

### Requirement: TUI Display

The TUI SHALL show MCPs in a navigable list with status indicators.

#### Scenario: Show installed vs available
Given MCPs are loaded
When TUI renders
Then installed MCPs show checked (☑), others show unchecked (☐)

#### Scenario: Group by tags
Given MCPs have tags
When TUI renders
Then tag sections are shown with expandable lists

#### Scenario: Pending changes indicator
Given user toggles an MCP
When TUI renders
Then pending install shows "(pending)", pending remove shows "(will remove)"

### Requirement: TUI Navigation

The TUI SHALL support keyboard-driven navigation and actions.

#### Scenario: Arrow key navigation
Given TUI is displayed
When user presses up/down or k/j
Then cursor moves through list

#### Scenario: Space toggles selection
Given cursor is on an MCP
When user presses space
Then MCP's checked state toggles

#### Scenario: Enter applies changes
Given pending changes exist
When user presses enter
Then changes are written to adapter and TUI exits

#### Scenario: Search mode
Given TUI is displayed
When user presses /
Then search input is focused and typing filters list

### Requirement: CLI List Command

The module SHALL provide a command to list MCPs from the command line.

#### Scenario: List installed only
Given `cly mcp list`
When command runs
Then only installed MCPs are shown

#### Scenario: List all with flag
Given `cly mcp list --all`
When command runs
Then both installed and available MCPs are shown

### Requirement: CLI Switch Command

The module SHALL provide a command to toggle MCPs from the command line.

#### Scenario: Toggle single MCP
Given `cly mcp switch github`
When command runs
Then github MCP toggles between installed/uninstalled

#### Scenario: Force enable
Given `cly mcp switch github --on`
When command runs
Then github is installed (no-op if already installed)

#### Scenario: Force disable
Given `cly mcp switch github --off`
When command runs
Then github is uninstalled (no-op if not installed)

#### Scenario: Switch by preset
Given `cly mcp switch -p webdev`
When command runs
Then all MCPs in webdev preset are toggled

#### Scenario: Switch by tag
Given `cly mcp switch -t cloud --on`
When command runs
Then all MCPs with cloud tag are enabled

### Requirement: CLI Add Command

The module SHALL provide a command to add MCPs to the source catalog.

#### Scenario: Add stdio MCP
Given `cly mcp add github npx -y @anthropic/mcp-github`
When command runs
Then MCP is added to `~/.config/mcpcli/mcps/custom.yaml`

#### Scenario: Add with transport flag
Given `cly mcp add --transport http sentry https://mcp.sentry.dev/mcp`
When command runs
Then HTTP MCP is added to catalog

#### Scenario: Add with env and tags
Given `cly mcp add --env API_KEY=xxx --tags cloud,aws myaws npx aws-mcp`
When command runs
Then MCP is added with env vars and tags

### Requirement: CLI Remove Command

The module SHALL provide a command to remove MCPs from the source catalog.

#### Scenario: Remove existing MCP
Given `cly mcp remove github`
When command runs
Then MCP is removed from source file

#### Scenario: Remove non-existent MCP
Given `cly mcp remove nonexistent`
When command runs
Then error is shown with suggestion

### Requirement: CLI Context Command

The module SHALL provide a command to show and set the default context.

#### Scenario: Show current context
Given `cly mcp context --show`
When command runs
Then current AI tool and scope are displayed

#### Scenario: Show config path
Given `cly mcp context --which`
When command runs
Then path to config file is displayed

#### Scenario: Set context directly
Given `cly mcp context cursor:project`
When command runs
Then default context is updated

#### Scenario: Interactive context selection
Given `cly mcp context` with no args
When command runs
Then interactive menu is shown

### Requirement: CLI Doctor Command

The module SHALL provide a command to check system health.

#### Scenario: Basic health check
Given `cly mcp doctor`
When command runs
Then config paths, adapters, and catalog are checked

#### Scenario: Detailed validation
Given `cly mcp doctor --validate`
When command runs
Then all contexts are validated with detailed output

### Requirement: Shell Completions

The module SHALL generate shell completions for bash, zsh, and fish.

#### Scenario: Generate bash completion
Given `cly mcp completion bash`
When command runs
Then bash completion script is output

#### Scenario: Generate zsh completion
Given `cly mcp completion zsh`
When command runs
Then zsh completion script is output

#### Scenario: Generate fish completion
Given `cly mcp completion fish`
When command runs
Then fish completion script is output

### Requirement: TUI Context Switcher

The TUI SHALL provide an overlay to switch between AI tools and scopes.

#### Scenario: Open context switcher
Given TUI is displayed
When user presses c
Then context switcher overlay appears

#### Scenario: Select new context
Given context switcher is open
When user selects cursor:project and presses enter
Then context switches and MCPs reload

### Requirement: TUI Section Collapsing

The TUI SHALL allow collapsing sections with number keys.

#### Scenario: Toggle validation section
Given TUI is displayed
When user presses 0
Then validation section collapses/expands

#### Scenario: Toggle presets section
Given TUI is displayed
When user presses 1
Then presets section collapses/expands

#### Scenario: Toggle tags section
Given TUI is displayed
When user presses 2
Then tags section collapses/expands

### Requirement: Validation Display

The TUI SHALL display validation issues when present.

#### Scenario: Show validation errors
Given catalog has parsing errors
When TUI renders
Then validation section shows errors with red indicator

#### Scenario: Show validation warnings
Given config has warnings
When TUI renders
Then validation section shows warnings with yellow indicator

#### Scenario: Refresh validation
Given TUI is displayed
When user presses v
Then validation is re-run and display updates
