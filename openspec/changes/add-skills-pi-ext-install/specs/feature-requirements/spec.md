# Feature Requirements: cly skills install + cly pi extensions install

## Commands

### `cly skills install`
Install cly-bundled AI agent skills to the local filesystem.

**Signature:**
```
cly skills install [--target <dir>] [--dry-run]
```

**Flags:**
- `--target <dir>` — Override install destination. Default: `~/.agents/skills/`.
- `--dry-run` — Print what would be written without touching the filesystem.

**Behavior:**
- For every file in the embedded skills tree (`modules/skills/embedded/...`), write it to the corresponding path under `--target`, creating directories as needed.
- Overwrite existing files unconditionally.
- Print one status line per file to stdout:
  - `wrote <abs-path>` for new files
  - `overwrote <abs-path>` for replaced files
  - `would write <abs-path>` under `--dry-run`
- Exit 0 on success; non-zero with a clear error on filesystem failure.

**Initial content:**
- `agents-session/SKILL.md` — session rules for AI agents (clean-state startup, file ownership, safe git, communication/production safety, tmp hygiene).

### `cly pi extensions install`
Install cly-bundled pi TUI extensions.

**Signature:**
```
cly pi extensions install [--target <dir>] [--dry-run]
```

**Flags:**
- `--target <dir>` — Override install destination. Default: `~/.pi/agent/extensions/`.
- `--dry-run` — Print what would be written without touching the filesystem.

**Behavior:**
Same as `cly skills install`, but the embedded tree is `modules/pi/embedded/...` and the default target differs.

**Initial content:**
- `save/` — TypeScript pi extension providing the `/save` slash command.

## Pi extension: `/save`

Registers a `/save` slash command in the pi TUI that calls `cly gs save`.

**Invocation forms:**
| Input                                             | name                     | description                  |
|---------------------------------------------------|--------------------------|------------------------------|
| `/save`                                           | default from code        | default from code            |
| `/save some name here`                            | `some name here`         | default from code            |
| `/save description="the description"`             | default from code        | `the description`            |
| `/save some name here description="the description"` | `some name here`      | `the description`            |

**Argument parsing rules:**
- Parser extracts key-value pairs matching `key="value"` (quoted) or `key=bareword` from the argument string.
- Remaining text (after kv removal) is trimmed; if non-empty, it becomes the `name` override.
- Currently supported keys: `description`. Unknown keys are ignored (forward compatible).

**Prefill rules (when no override):**
- `id` — always computed in extension code (deterministic); never taken from user input.
- `name` — default computed in extension code.
- `description` — default computed in extension code.

**Output:**
The extension shells out to `cly gs save` and surfaces its stdout/stderr in the pi TUI.
