# CLY Quick Reference

## Development

```bash
go run main.go <command>      # Run command in dev
task build                    # Build binary
task test                     # Run tests
```

## Common Commands

```bash
cly uuid                      # Generate UUIDs
cly helpy / hy                # View help (this file or ~/DotFiles/HELP.md)
cly helpy -i / hy -i          # Open Pi with DotFiles + cly context
cly helpy -i --ai claude      # Override AI CLI (e.g. claude, opencode)
cly demo <name>               # Run TUI demos
cly scraper browser           # Launch scraper browser
cly scraper aliexpress --url  # Scrape AliExpress
cly config show               # View config
cly config set <key> <value>  # Set config value
cly mcp list                  # List MCP servers
```

## Architecture

Modular CLI with Cobra + Bubbletea + Viper. Each module in `modules/` registers itself in `cmd/root.go`.

Config: `~/.config/cly/config.yaml`
Data: `~/.local/share/cly/`

## Adding Modules

1. Create `modules/<name>/cmd.go` with `Register(parent *cobra.Command)`
2. Add to `cmd/root.go` init()
3. Use skills: `/add-module`, `/testing`, `/charm-stack`
