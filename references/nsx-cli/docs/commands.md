## NSX CLI Command Reference

This document lists all available commands, flags, and examples for the NSX CLI.

Tip: All commands support inline help. Append `-h` or `--help` to any command for details.

```bash
nsx --help
nsx <command> --help
nsx <group> <subcommand> --help
```

### Global flags

- `--skin, -s` (default: `betdev`): Select the target skin/system.
  - Allowed values: `betdev`, `betnacional`, `mrjackbets`, `mundialbet`
- `--debug, -d`: Enable verbose debugging output.

Example:

```bash
nsx -s betnacional --debug <command> ...
```

## Command tree

```
nsx
├── ai
│   └── sync
├── customer
│   ├── recover-account <customerID>
│   └── generate-token
├── sre
│   ├── db
│   │   └── long-running
│   └── incidentsio
├── login
├── project
│   └── build
├── secrets
├── update
└── version
```

## Top-level commands

### login
Authenticate with Google APIs to enable features that read/write Google Drive and Sheets.

```bash
nsx login
```

Behavior:
- Opens a browser for OAuth2, stores credentials locally under your NSX config directory.

### project
Tools for managing NSX projects.

Subcommands:

#### project build
Scaffold a new service from the NSX project builder.

Flags:
- `--name, -n` (string): Name of the project. Required for meaningful output.
- `--team, -t` (string): Team responsible for the project.

Examples:
```bash
nsx project build --name account-service --team sre
nsx -s betnacional project build -n payments -t customer
```

### secrets
Interactive AWS Secrets Manager selector with dynamic filtering.

Flags:
- `--output, -o` (string): Save selected secret (or JSON) to a file.
- `--region, -r` (string, default: `us-east-1`): AWS region.

Examples:
```bash
nsx secrets
nsx secrets -r us-west-2
nsx secrets -o secret.json
```

Features include filtering, viewing, copying keys/values to the clipboard, and saving to file.

### update
Update the NSX CLI to the latest or a specific version.

Flags:
- `--force, -f`: Force update even if already on the latest version (useful for dev builds).
- `--version, -v` (string): Target version (default: latest).

Examples:
```bash
nsx update
nsx update -v 1.2.3
nsx update --force
```

### version
Display the current NSX CLI version and check if a newer version is available.

```bash
nsx version
```

## Team commands

### ai
AI tools for NSX.

#### ai sync
Sync `.cursor/rules` from the `NSXBet/cursor-rules` repository to the current project.

Environment:
- Requires `GHA_PAT` environment variable set to a GitHub token with access to the repo.

Examples:
```bash
export GHA_PAT=ghp_your_token_here
nsx ai sync
```

### customer
Customer team tools.

#### customer recover-account <customerID>
Recover a self-deleted customer account.

Arguments:
- `<customerID>`: The numeric customer ID to recover. Required.

Example:
```bash
nsx customer recover-account 123456
```

Notes:
- Uses the internal Customer Service API; host and token are resolved from AWS Secrets based on `--skin`.

#### customer generate-token
Interactive TUI to generate a short-lived JWT token for a customer.

Behavior:
- Loads configuration from AWS Secrets (based on `--skin`).
- Prompts for Customer ID and token duration (e.g., `30d`, `24h`, `1h30m`).
- Copies the generated token to your clipboard and displays it.

Example:
```bash
nsx customer generate-token
```

### sre
SRE tools for NSX.

#### sre db long-running
List long-running queries for a given RDS instance. Supports MySQL, PostgreSQL, and Aurora variants.

Flags:
- `--rds-instance, -i` (string, required): RDS instance identifier to connect to.
- `--min-duration, -m` (int, default: 30): Minimum query duration in seconds (minimum allowed: 10).
- `--database, -n` (string): Specific database name to connect to.
- `--region, -r` (string, default: `us-east-1`): AWS region for the RDS instance.

Examples:
```bash
nsx sre db long-running -i my-rds-instance
nsx sre db long-running -i my-rds-instance -m 60
nsx sre db long-running -i my-rds-instance -n app_db -r us-west-2 --debug
```

See also: `docs/sre-db-long-running.md` for additional details and troubleshooting.

Prerequisites:
- AWS CLI authenticated with the `DeveloperBase` role and proper RDS permissions.

#### sre incidentsio
Fetch and display incidents and follow-ups from the incident.io API.

Flags:
- `--start-date` (YYYY-MM-DD): Start date; defaults to 7 days ago if omitted.
- `--end-date` (YYYY-MM-DD): End date; defaults to today if omitted.
- `--format` (string, default: `markdown`): One of `json`, `markdown`, `charm`, `text`.
- `--status` (string): Filter incidents by status.

Environment:
- Requires `INCIDENTSIO_API_KEY` to be set.

Examples:
```bash
export INCIDENTSIO_API_KEY=xxx
nsx sre incidentsio --start-date 2025-01-01 --end-date 2025-01-31 --format json
nsx sre incidentsio --status closed
```


