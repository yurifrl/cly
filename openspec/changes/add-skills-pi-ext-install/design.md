# add-skills-pi-ext-install - Technical Design

## Architecture Changes

Two new self-registering cly modules under `modules/`:

```
modules/skills/
├── cmd.go                      # Register() + `install` subcommand
├── skills.go                   # Install logic (walk embed.FS, write tree)
├── embedded/                   # //go:embed embedded
│   └── agents-session/
│       └── SKILL.md
└── skills_test.go

modules/pi/
├── cmd.go                      # Register() + `extensions` parent + `install` subcommand
├── extensions.go               # Install logic (same pattern as skills)
├── embedded/                   # //go:embed embedded
│   └── save/
│       ├── package.json
│       ├── tsconfig.json
│       └── src/index.ts        # /save slash command
└── pi_test.go
```

Registered in `cmd/root.go` init() with two new lines:
```go
skills.Register(RootCmd)
pi.Register(RootCmd)
```

Both modules follow the "parent with subcommands" pattern from `AGENTS.md`.

## Data Model Changes
None. No persistent state; both installers are pure filesystem operations.

## API Changes

### `cly skills install`
```
cly skills install [--target <dir>] [--dry-run]
```
- Default target: `~/.agents/skills/`
- Walks embedded FS; for each file under `embedded/<skill>/...`:
  - `MkdirAll` target subdir
  - Overwrite destination file with embedded bytes
  - Print one line: `wrote <abs-path>` | `overwrote <abs-path>` | `would write <abs-path>` (dry-run)
- Exit 0 on success; non-zero with clear error on FS failure.

### `cly pi extensions install`
```
cly pi extensions install [--target <dir>] [--dry-run]
```
- Default target: `~/.pi/agent/extensions/`
- Same walk/overwrite/log behavior as skills.

### Pi extension: `/save`
- TypeScript extension registered with pi as a slash command.
- Handler parses the slash argument string:
  1. Split off kv pairs matching `<key>="<value>"` (quoted) or `<key>=<bareword>`.
  2. Remaining text is positional → `name` override.
- Resolves final values:
  - `id` — deterministic in TS (e.g. ulid / timestamp; exact algorithm TBD at implementation time)
  - `name` — positional override, else default from code
  - `description` — `description="..."` kv override, else default from code
- Invokes: `cly gs save --id <id> --name <name> --description <description>` (exact flags to match `cly gs save`'s contract).

## UI/UX Changes
- New CLI subtrees: `cly skills ...`, `cly pi extensions ...` with standard cobra `--help`.
- New pi slash command: `/save`. Output: whatever `cly gs save` prints, surfaced in the pi TUI.

## Testing Strategy

**Installers (Go, integration-style):**
- Point `--target` at a `t.TempDir()`.
- Run install, assert:
  - Every embedded file is present at the expected path.
  - Content matches embedded bytes.
  - Stdout contains a line per file.
- Re-run install on same target, assert overwrite behavior + log lines say `overwrote`.
- Run with `--dry-run`, assert no files written + log lines say `would write`.

**Pi extension (manual to start):**
- Install extension; verify pi loads it.
- Invoke `/save`, `/save some name`, `/save name here description="foo"`, assert `cly gs save` is called with expected args (via a shim script on `$PATH` during manual test).

## Performance Impact
Negligible. Installers are one-shot file copies of small text files; extension is a thin wrapper that shells out to `cly`.

## Security Considerations
- Installers write only under `--target` (default: user-owned dirs under `$HOME`). No privilege escalation, no network.
- Pi extension shells out to `cly` binary found on `$PATH`. Standard PATH-hijack caveats; acceptable for a user-local CLI tool.
- No secrets handled.

## Deployment Notes
- Embed content is baked at build time. Rebuilding cly is sufficient to ship updated skills/extensions.
- Dotfiles integration (user-side):
  ```
  @once cly-skills  -- cly skills install
  @once cly-pi-ext  -- cly pi extensions install
  ```
- No migrations. Users re-running install will see `overwrote` lines for any drift.
