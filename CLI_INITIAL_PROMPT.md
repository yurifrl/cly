First of all, before anything, this is most of all a tech demo for charm land libraries

Create a golang cli using https://charm.land/libs/ this is a cli to add random scripts, it should be modular  ~/Workdir/Nsx/nsx-cli/ is modular, I like they way you can add new teamsand commands easily. I don't like that it issues toml for config, config should be yaml using https://github.com/spf13/viper, charm is a tui interface for cli we should use cobra

charm libraries:
    - tui
    - everytime i need to list something in the terminal
    - Need a full terminal interface
    - Simple or Complex list display
    - Checkboxes pick from selections
    - Visualization
    - $AI(Need to add more example here after unsderstanding what can be done with charm tui)
viper: 
    - read and write yaml config files
    - support for env variables
cobra:
    - CLI application framework
    - Command and subcommand structure
    - Argument parsing
    - Help generation
    - cli ONLY


Modularity

The cli modularity is represented by being able to add new commands in isolation without modifying the core application code. Each command can be developed as a separate package or module, allowing for easy addition, removal, or updating of commands.

Each module is made in such a maner it could be easily extracted. By relying on interfaces and abstractions, the core application can interact with these modules without needing to know their internal workings. Each module has 1 entry point and everything is passed as parameters or via interfaces allowing loose coupling.

First implementation

First implementation is a charm tech demo make a complex demo of https://github.com/charmbracelet/bubbletea and https://github.com/charmbracelet/huh and https://github.com/charmbracelet/lipgloss and https://github.com/charmbracelet/bubbles 
Need to understand this libraries to understand what will be done
