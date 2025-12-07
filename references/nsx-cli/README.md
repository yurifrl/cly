# NSX CLI

A powerful command-line interface for NSX, designed to streamline workflows and improve productivity for NSX team members.

```
888b    888  .d8888b. Y88b   d88P       .d8888b.  888      8888888
8888b   888 d88P  Y88b Y88b d88P       d88P  Y88b 888        888
88888b  888 Y88b.       Y88o88P        888    888 888        888
888Y88b 888  "Y888b.     Y888P         888        888        888
888 Y88b888     "Y88b.   d888b         888        888        888
888  Y88888       "888  d88888b        888    888 888        888
888   Y8888 Y88b  d88P d88P Y88b       Y88b  d88P 888        888
888    Y888  "Y8888P" d88P   Y88b       "Y8888P"  88888888 8888888
```

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Shell Completion](#shell-completion)
- [Usage](#usage)
- [Command Reference](#command-reference)
- [Adding New Commands](#adding-new-commands)
- [Creating a Team Folder](#creating-a-team-folder)
- [Shared Packages](#shared-packages)
- [Configuration](#configuration)

## Overview

NSX CLI is a Go-based tool that provides a unified interface for interacting with NSX services, resources, and environments. Built with extensibility in mind, it supports team-specific commands while maintaining a consistent developer experience.

Key features:

- **Cross-platform**: Works on macOS, Linux, and Windows
- **Team-based extensibility**: Easily add team-specific commands
- **Shared utilities**: Common functionality available across all commands
- **Modern configuration**: Flexible config patterns for different environments
- **Authentication**: Integrated Google login for secure access

## Project Structure

```
nsx-cli/
├── cmd/                  # Command implementations
│   ├── root.go           # Root command setup
│   └── ...               # Individual commands
├── shared/               # Shared utilities
│   ├── auth/             # Authentication utilities
│   ├── config/           # Configuration management
│   └── ...               # Other shared components
├── team/                 # Team-specific commands
│   └── ...               # Individual team packages
├── main.go               # Entry point
├── .goreleaser.yaml      # GoReleaser configuration
└── ...                   # Other configuration files
```

## Installation

### macOS and Linux

```bash
curl -fsSL https://nsx-cli-proxy.nsx.services/main/install.sh | sh
```

### Windows

```powershell
iwr -useb https://nsx-cli-proxy.nsx.services/main/install.ps1 | iex
```

### Manual Installation

If you prefer to manually install or build from source:

> The binary name in this case will be nsx-cli, I strongly recomend you to create an alias that shotens this to nsx
> If you go this route, and you intend to write some google integrated script, you will need to have a google credential to be defined in the enviroriment. You can create a `.env` file and add the following

```
GOOGLE_CLIENT_ID=something
GOOGLE_CLIENT_SECRET=somethinsgomething
```

1. Install directly using Go:

   ```bash
   go install github.com/NSXBet/nsx-cli@latest
   ```

> This will install the binary to your `$GOPATH/bin` directory (or `$GOBIN` if set). Make sure this directory is in your PATH.

2. Verify the installation:
   ```bash
   nsx version
   ```

Alternatively, to build from a local copy:

1. Clone the repository:

   ```bash
   git clone https://github.com/NSXBet/nsx-cli.git
   cd nsx-cli
   ```

2. Install the binary:

   ```bash
   go install .
   ```

## Shell Completion

NSX provides built-in support for shell completion, which will help you navigate commands and options more efficiently.

### Setting up Shell Completion

You can enable shell completion for your preferred shell by running:

```bash
# For bash
nsx completion bash >> ~/.bashrc

# For zsh
nsx completion zsh >> ~/.zshrc

# For fish
nsx completion fish >> ~/.config/fish/completions/nsx.fish

# For PowerShell
nsx completion powershell >> $PROFILE
```

This command will generate the appropriate completion script for your shell and add it to your shell configuration file. After adding the completion script, restart your terminal or source your configuration file to enable completions:

```bash
# For bash
source ~/.bashrc

# For zsh
source ~/.zshrc

# For fish
source ~/.config/fish/config.fish
```

With completion enabled, you can now use <TAB> to see available commands, subcommands, and flags as you type.

## Usage

Once installed, you can run NSX commands:

```bash
# Get help
nsx --help

# List available commands
nsx help

# Run a specific command
nsx <command> [args]

# Authenticate with Google
nsx login
```

## Export Gobin Path for asdf
```
export PATH=$PATH:$GOBIN
```
After, close your terminal and open again.

## Command Reference

For a complete list of commands, flags, and examples, see the Command Reference:

- [docs/commands.md](docs/commands.md)

## Adding New Commands

NSX CLI uses a modular structure based on Cobra for command-line parsing. To add a new command:

1. Create a new file in the `cmd` directory:

```go
// cmd/your_command.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(yourCmd)
}

var yourCmd = &cobra.Command{
	Use:   "your-command",
	Short: "Brief description of your command",
	Long:  `A longer description for your command.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Your command logic here
		fmt.Println("Your command executed!")
	},
}
```

2. Test your command:

```bash
go run main.go your-command
```

3. Add any flags or subcommands as needed:

```go
var flagName string

func init() {
	rootCmd.AddCommand(yourCmd)

	// Add flags
	yourCmd.Flags().StringVarP(&flagName, "flag-name", "f", "default", "Description of flag")
}
```

## Command Structure

NSX organizes commands in a hierarchical structure:

```
nsx
├── global commands       # Available to all users
│   ├── login             # Authenticate with NSX services
│   ├── config            # Manage configuration
│   └── ...               # Other global commands
├── team commands         # Team-specific functionality
│   ├── team1             # Commands for team1
│   │   ├── command1
│   │   └── command2
│   ├── team2             # Commands for team2
│   └── ...
└── ...
```

## Creating a Team Folder

Team folders allow different teams to maintain their own set of commands while sharing the NSX architecture. This modular approach helps keep code organized and enables teams to work independently.

1. Create a new directory in the `team` folder with your team name:

```bash
mkdir -p team/yourteam
```

2. Create a new Go file for your team's main functionality:

```go
// team/yourteam/yourteam.go
package yourteam

import (
	"github.com/spf13/cobra"
)

// GetCommands returns the collection of commands for this team
func GetCommands() []*cobra.Command {
	return []*cobra.Command{
		MyTeamCmd(),
		// Add more team-specific commands here
	}
}

// MyTeamCmd creates the main command for your team
func MyTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yourteam",
		Short: "YourTeam specific commands",
		Long:  `Commands specific to YourTeam operations and workflows.`,
	}

	// Add subcommands
	cmd.AddCommand(yourTeamSubCommand())

	return cmd
}

// Example subcommand
func yourTeamSubCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "subcommand",
		Short: "A subcommand for your team",
		Run: func(cmd *cobra.Command, args []string) {
			// Subcommand logic here
		},
	}
}
```

3. Register your team's commands in the main command structure by updating `cmd/root.go` to import and register your team:

```go
// In cmd/root.go
import (
	// Other imports
	"github.com/NSXBet/nsx/team/yourteam"
)

func init() {
	// Register your team's commands
	for _, cmd := range yourteam.GetCommands() {
		rootCmd.AddCommand(cmd)
	}
}
```

When your team folder is properly set up, users can access your team's commands via:

```bash
nsx yourteam [subcommand]
```

## Shared Packages

The `shared` directory contains common utilities and packages that can be used across different commands and teams. This helps maintain consistency and reduces code duplication.

Key shared packages include:

- `shared/config`: Configuration management for user settings
- `shared/googlex`: A set of functions to use google sessions tools (read/write from sheets from instance)
- `shared/interact`: A package to help you write messages for the user like info, success, error, warn, debug
- `shared/skin`: A package to manage the global flag `--skin/-s`

> We encorage you to use those libraries to make the CLI subcommands consistent. You can create subcommands to behaive like TUI using stuff like [Charmbracelet stack](https://github.com/charmbracelet), but if is it just to write messages in a script form, use interact, espetially for the debug stuff.

To use a shared package in your command:

```go
import (
    "os"

	"github.com/NSXBet/nsx/shared/interact"
	"github.com/NSXBet/nsx/shared/skin"
)

func yourFunction() {
    currentSkin = skin.GetSkin()
    username = os.Getenv("USER")
    interact.Info("Hey %s, what you want to do in the skin %s", username, currentSkin)
}
```

### Benefits of Using Shared Packages

- **Consistency**: All teams use the same underlying components
- **Maintenance**: Bug fixes in shared code benefit all commands
- **Efficiency**: Reduces duplication and development time
- **Best Practices**: Enforces standardized approaches to common tasks

## Configuration

NSX-CLI uses a strategy to store sensible data in a way to avoid security risks. The configuration is stored at the 1password and kept encrypted in the path `~/.config/nsx/`. All config files must be using the `TOML` format.

> If you are testing things locally, you can change the config folder using the enviroriment var `NSX_CONFIG_PATH` to your prefered destination

**Example of usage**

```go
import (
	"github.com/NSXBet/nsx-cli/shared/config"
	"github.com/NSXBet/nsx-cli/shared/skin"
)

const (
	Registry       = "op://TEAM Voult/item/nsx_cli_team_config"
	ConfigFilename = "nsx_some_team_config.toml"
)

func Load() (*Config, error) {
	return config.Load[Config](ConfigFilename)
}

type Config struct {
	someString     string      `toml:"some_string"`
	someInt        int         `toml:"some_int"`
	someSlice      []any       `toml:"some_slice"`
	someMap        map[any]any `toml:"some_map"`
	someCustomType CustomType  `toml:"some_custom_type"`
}

```

The load function will search fo the `nsx_some_team_config.toml` in the config path decript and parse the file.

An example of the toml file would be this:

```toml
some_string = 'test'
some_int = 4
some_slice = [{}, 'orange', 4]

[some_map.example_a]
    test = 455
    something = 'something'

[some_map.example_b]
    another_thing = 54.3
    some_char = 'c'

[some_custom_type]
    ....

```
