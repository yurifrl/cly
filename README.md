# CLY - Command Line Utilities

Modular Go CLI for day-to-day utilities with beautiful TUI interfaces powered by Charm libraries.

## Features

- **Claude Code Statusline** - Context window, model, cost display for Claude Code sessions
- **UUID Generator** - Interactive UUID generation (v4, v7, multiple)
- **AliExpress Scraper** - Browser automation for product data extraction
- **48 TUI Demos** - Complete Bubbletea component showcase
- **Modular Architecture** - Zero-coupling design, easy to extend
- **Single Binary** - No runtime dependencies

## Installation

### Quick Install (Recommended)
```bash
curl -sSL https://raw.githubusercontent.com/yurifrl/cly/main/install.sh | bash
```

The script installs to `~/.local/bin/cly` by default. Set `INSTALL_DIR` to customize:
```bash
INSTALL_DIR=/usr/local/bin ./install.sh
```

### Via go install
```bash
go install github.com/yurifrl/cly@latest
```

### Build from source
```bash
git clone https://github.com/yurifrl/cly
cd cly
go run main.go --help
```

## Usage

### UUID Generator
```bash
cly uuid
# Interactive selection: v4 (random), v7 (time-ordered), multiple (5x)
```

### AliExpress Scraper
```bash
# Launch persistent browser (solve CAPTCHA once)
cly scraper browser

# In another terminal, scrape products
cly scraper aliexpress --url 1005003618976317
cly scraper aliexpress --url "1005003618976317,1005010081760632"
cly scraper aliexpress -f products.txt
```

### TUI Component Demos
```bash
cly demo --help           # List all 48 demos
cly demo chat             # Chat room (textarea + viewport)
cly demo spinner          # Animated spinner
cly demo table            # Data table with 100 cities
cly demo list-simple      # Selection list
cly demo progress-static  # Progress bar
cly demo file-picker      # File picker
```

## Commands

### Utilities

| Command | Description |
|---------|-------------|
| `uuid` | Generate UUIDs interactively (v4, v7, multiple) |
| `scraper` | Web scraping with browser automation |
| `scraper browser` | Launch persistent browser for scraping |
| `scraper aliexpress` | Scrape AliExpress product data |
| `skills install [name...]` | Install cly-bundled AI agent skills (default `~/.agents/skills/`). Cherry-pick by name; default is all. |
| `pi extensions install` | Install cly-bundled pi extensions (default `~/.pi/agent/extensions/`). Currently ships `pi-cly` with a `/save` slash command. |

Dotfiles integration:

```
@once cly-skills -- cly skills install
@once cly-pi-ext -- cly pi extensions install
```

### Demos (48 Total)

The `demo` namespace contains all 48 official Bubbletea examples:

**Core Components:**
- `chat`, `spinner`, `table`, `list-simple`, `list-default`, `list-fancy`
- `textinput`, `textarea`, `textinputs`
- `progress-static`, `progress-animated`, `progress-download`

**Forms & Pickers:**
- `credit-card-form`, `file-picker`, `autocomplete`

**Layout & Views:**
- `pager`, `paginator`, `viewport`, `tabs`, `split-editors`
- `views`, `composable-views`

**Time & Animation:**
- `timer`, `stopwatch`, `spinners`

**Advanced:**
- `mouse`, `focus-blur`, `prevent-quit`, `window-size`
- `exec`, `pipe`, `suspend`, `tui-daemon-combo`
- `send-msg`, `sequence`, `result`, `realtime`
- `debounce`, `eyes`, `cellbuffer`, `glamour`
- `altscreen-toggle`, `fullscreen`, `set-window-title`
- `simple`, `help`, `http`, `package-manager`, `table-resize`

Run `cly demo --help` to see all available demos.

## Architecture

Modular design with clean command registration:

```
cly/
├── main.go              # Entry point
├── cmd/
│   └── root.go          # Root command, module registration
├── modules/
│   ├── uuid/            # UUID utility
│   │   ├── cmd.go       # Command registration
│   │   └── uuid.go      # Implementation
│   └── demo/            # Demo namespace (48 subcommands)
│       ├── cmd.go       # Demo parent + registrations
│       ├── chat/
│       ├── spinner/
│       └── ... (46 more)
└── pkg/
    └── style/           # Shared Lipgloss styles
        └── theme.go
```

**Key principles:**
- Each module is self-contained
- Zero coupling between modules
- Single registration point in `cmd/root.go`
- Adding command = create directory + add one line to root

## Adding a Module

See [`docs/adding-modules.md`](docs/adding-modules.md) for detailed template and instructions.

Quick steps:
1. Create `modules/<name>/` directory
2. Copy pattern from existing module
3. Adapt for your use case
4. Register in `cmd/root.go` init()

Or use the Claude Code skill: Just ask Claude to "create a demo module for X"

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework (Elm Architecture)
- [Bubbles](https://github.com/charmbracelet/bubbles) - Pre-built TUI components
- [Huh](https://github.com/charmbracelet/huh) - Forms and prompts
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout

## Development

```bash
# Run in development
go run main.go demo chat

# Build binary
go build

# Run tests (when implemented)
go test ./...
```
