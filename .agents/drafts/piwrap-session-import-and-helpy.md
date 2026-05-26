# piwrap session import + --helpy

Consolidates the full design discussion from the 2026-05-22 session: pi session pre-creation, `--sety` typed flag namespace for piwrap, session import/fork pipeline, conflict + quarantine handling, `cp` + bytes-replace fork strategy, `--helpy` registry-driven cheat sheet (text + JSON), schema introspection, dry-run, and AI-friendly CLI principles applied throughout.

> Note: This is a draft to organize ideas and scope before implementation.

## Goal

Two related features on top of the existing piwrap (`-n / --name` already pins a stable pi session file per cwd):

1. **Import an existing pi session** into the current cwd under a chosen name. User passes a UUID (or prefix), piwrap resolves the source JSONL, forks it (new session id), writes it to the named target path, and handles conflicts safely (move existing to a recoverable trash dir, never `rm`).
2. **Surface cly's custom-on-top-of-pi flags** through a `--helpy` cheat sheet so users and agents discover them without grepping source. Generated from a registry, no hand-written drift.

Both features follow AI-friendly CLI principles: typed input, structured JSON output, dry-run, schema introspection, and structured errors on stdout with hints on stderr.

## Background — what already exists in piwrap

`modules/piwrap` intercepts `--name`/`-n`, sets `$CLY_SESSION_NAME`, renames the cmux tab, and (shipped earlier in this session) computes a stable session path:

```
~/.pi/agent/sessions/<encodeCwd(cwd)>/<prefix>-<kebab(name)>.jsonl
```

Where:
- `encodeCwd("/Users/yuri/Workdir/Yuri/cly")` -> `--Users-yuri-Workdir-Yuri-cly--` (strip leading `/`, replace `/` with `-`, wrap with `--...--`).
- `<prefix>` from `modules.piwrap.session_file_name_prefix` (default `cly`).
- `<kebab(name)>` is ASCII lowercase, non-alphanumeric runs collapse to `-`, trimmed.

Pi's `--session` flag accepts either an existing session ID (lookup-only — bare unknown UUIDs error with `No session found matching '<id>'`) **or a path** (creates the file on miss). Piwrap injects the path form, so pi creates the file the first time and reopens it on subsequent runs. The session ID inside the JSONL is pi-generated; the filename is the stable handle.

## Patterns considered for nested key=value (background)

Common Go CLI conventions surveyed during the discussion, ordered by fit:

1. **Repeatable `--set key=value` (dotted).** Helm, kubectl, terraform-style. Pflag-friendly. Chosen and narrowed (see Decisions).
2. **Inline JSON / YAML.** `--json '{...}'`. Best for AI agents (single payload, schema-validated). Recommended by ai-friendly-cli skill (Principle 1) over many flags. Not adopted here, called out as a future second surface.
3. **Repeated `-o key=value`.** SSH-style. Same semantics as `--set`.
4. **Env vars with prefix.** `CLY_FOO__A=bar`. Already supported by Viper.
5. **Comma-separated.** Helm fallback. Trips on commas in values.
6. **File overlay.** `--config foo.yaml`, last-wins merge. Cleanest for many values; out of scope.

Decision: pattern 1 only, narrowed to a fixed key set scoped to piwrap.

## Architecture

`piwrap` parses `--sety key=value` (repeatable, **fixed keys**) and `-n name` before forwarding to `pi`. When `session_import.id` is set, piwrap runs an import pipeline:

```
resolve source (uuid|prefix|path)
  -> handle conflict at target (override -> quarantine, else fail)
  -> cp src target.tmp
  -> bytes.ReplaceAll(buf, oldSessionID, newSessionID)
  -> fsync + rename to target
  -> inject --session <target> into args forwarded to pi
```

`--helpy` short-circuits before any subcommand: walks a static registry, renders text (lipgloss) or JSON, exits 0.

```
+----------------------------+
|       cmd/root.go          |  --helpy?  -> render + exit
|       cobra dispatch       |
+----------------------------+
              |
              v
+----------------------------+
|     modules/piwrap         |  -n, --sety
|  - extractName             |
|  - parseSety (fixed keys)  |
|  - resolveSource           |
|  - quarantineIfConflict    |
|  - forkSession             |
|  - inject --session <path> |
+----------------------------+
              |
              v
            pi exec
```

## CLI surface

All flags below live on the **`cly pi`** subcommand (alias `cly p`). They are NOT global cly flags. Piwrap has `DisableFlagParsing: true`, so unknown flags pass straight to the underlying `pi` binary; piwrap only intercepts the names listed here.

```bash
# pre-create / reopen a named session (already shipped)
cly pi -n refactor-auth
cly pi --name "My Work" -p "summarize"
cly p -n refactor-auth                          # alias

# import (fork) an existing pi session into this cwd, name it via -n
cly pi -n bug-repro --sety session_import.id=019e5057
cly pi -n bug-repro --sety session_import.id=$UUID --sety session_import.override=true
cly pi -n bug-repro --sety session_import.id=$UUID --dry-run

# value with no coercion (rare; override is bool, but provided for symmetry)
cly pi -n foo --sety-string session_import.id=019e5057-19ae-7ddc-a9e2-42abd19c8053

# cheat sheet for cly pi
cly pi --helpy
cly pi --helpy -o json
```

`--sety` is **piwrap-only with fixed keys**, not a generic cly config setter and not a global cly flag. Two valid keys, period.

## Components

### piwrap

- **`modules/piwrap/sety.go`** — parse `--sety key=value`, repeatable. **Fixed key set:**
  - `session_import.id` (string: full UUID, >=8-char prefix, or absolute `.jsonl` path)
  - `session_import.override` (bool, coerced from `true|false|1|0`)

  Anything else: `SETY_UNKNOWN_KEY`. Bad format: `SETY_FORMAT`. Bad bool: `SETY_PARSE`.

- **`modules/piwrap/import.go`** — source resolution, conflict handling, quarantine.
- **`modules/piwrap/fork.go`** — `cp src target.tmp` -> grab old session id via regex on first JSONL line -> `bytes.ReplaceAll` -> atomic rename. Fresh UUID v7.
- **`modules/piwrap/piwrap.go`** — extend `Run` to parse `--sety`, validate `-n` required when `session_import.id` set, call import pipeline before exec'ing pi, inject `--session <target>`.
- **`modules/piwrap/cmd.go`** — currently `DisableFlagParsing: true` and forwards everything to `Run`. We continue to disable cobra flag parsing and intercept `--sety`, `--sety-string`, `--dry-run`, `--helpy` ourselves inside `Run`/the parser. Cobra never sees them. Reason: piwrap must remain a transparent passthrough to pi for everything else.

### config

- **`modules/config/config.yaml`**:
  ```yaml
  modules:
    piwrap:
      session_file_name_prefix: cly        # already shipped this session
      session_import:
        override: false
        quarantine_dir: ~/.local/share/cly/trash/pi-sessions
        search_scope: cwd                  # cwd | all
  ```

### helpy

- **`pkg/helpy/registry.go`** — typed `Entry` registry. Each piwrap flag registers itself at package init. Adding a flag without registering it = invisible in `--helpy` (review-catchable).

  ```go
  type Entry struct {
      Section     string
      Flags       []string
      Value       string
      Description string
      ConfigKeys  []string
      EnvVars     []string
      Requires    []string
      Errors      []string
      Examples    []string
  }
  ```

- **`pkg/helpy/render_text.go`** — lipgloss output via `pkg/style`. NO_COLOR or non-TTY -> strip styling.
- **`pkg/helpy/render_json.go`** — `encoding/json` over the same registry. Doubles as input to `cly schema cli` / `--describe`.
- **`cmd/root.go`** — untouched. `--helpy` lives on `cly pi`, not on root. (Earlier draft mistakenly wired it into root; corrected.)
- **`pkg/helpy/registry.go`** entries are owned by piwrap; there is no global cly cheat sheet in this draft. If we ever want one, root can add a sibling `--helpy` that walks a different registry partition.

## Behavior — state machine

```
input: name (-n), id (--sety session_import.id), override (--sety or config)
cwd:   $PWD
target = ~/.pi/agent/sessions/<encodeCwd(cwd)>/<prefix>-<kebab(name)>.jsonl

1. Validate
   - id set & name unset                 -> SETY_NAME_REQUIRED
   - id is uuid prefix < 8 chars         -> SETY_IMPORT_ID_TOO_SHORT
   - unknown --sety key                  -> SETY_UNKNOWN_KEY
   - bad bool for override               -> SETY_PARSE

2. Resolve source
   roots = (search_scope == cwd) ? [encodeCwd($PWD)] : [all dirs under sessions/]
   match filename '*<id>*' first; fall back to first-line sessionId scan
   - 0 candidates                        -> SETY_IMPORT_NOT_FOUND
   - >1 candidates                       -> SETY_IMPORT_AMBIGUOUS (list on stderr)
   - source absolute path                -> use as-is, skip lookup
   - source == target                    -> warn no-op, exit 0

3. Conflict at target
   - target absent                       -> step 4
   - target present, override=false      -> SETY_IMPORT_CONFLICT
   - target present, override=true       -> mv target -> quarantine, log on stderr

4. Fork
   - read oldSessionID from src first JSONL line
     regex: "sessionId"\s*:\s*"([0-9a-f-]{36})"
     - not found                         -> SETY_IMPORT_FAILED reason=no_session_id_in_source
   - newSessionID = uuid v7 (regen once if equal to old; else SETY_IMPORT_FAILED)
   - cp src target.tmp
   - bytes.ReplaceAll(buf, oldSessionID, newSessionID), write back, fsync
   - rename(target.tmp, target)
   - any error after step 3              -> remove target.tmp, restore quarantine, SETY_IMPORT_FAILED

5. Forward
   - inject --session <target> into args (only if user did not pass --session)
   - exec pi with rest of argv unchanged
   - cmux rename-tab + $CLY_SESSION_NAME (existing behavior)
```

## Conflict resolution / quarantine

Default dir: `~/.local/share/cly/trash/pi-sessions/` (sibling to existing `app.data_dir`). Configurable via `modules.piwrap.session_import.quarantine_dir`.

Files are **moved**, never `rm`'d (honors AGENTS.md "no `rm`" rule). Quarantined filename:

```
<UTC-timestamp>-<encodeCwd(cwd)>-<prefix>-<kebab(name)>.jsonl

# example
~/.local/share/cly/trash/pi-sessions/2026-05-22T15-47-09Z---Users-yuri-Workdir-Yuri-cly---cly-my-session.jsonl
```

Why a flat dir + encoded path in filename: easy to grep, easy to restore, no nested mkdir tax, no collisions.

On override=true, log on stderr:
```
moved existing session: ~/.pi/agent/sessions/.../cly-my-session.jsonl
                    -> ~/.local/share/cly/trash/pi-sessions/2026-05-22T...-cly-my-session.jsonl
```

Quarantine grows forever. Document a follow-up `cly piwrap gc --older-than 30d`; do not auto-delete.

## Source resolution

`session_import.id` accepts:
- full UUID: `019e5057-19ae-7ddc-a9e2-42abd19c8053`
- prefix UUID: `019e5057` (>=8 chars to avoid collisions; reject shorter)
- absolute path to `.jsonl` (escape hatch; skips lookup; search_scope ignored)

Lookup: scan candidate roots, match filename containing `<id>` first (cheap), then fall back to reading first JSONL line for `sessionId` and matching by prefix. >1 hit -> ambiguous error with candidate list on stderr.

## Fork strategy — cp + bytes.ReplaceAll

Chosen over JSONL streaming rewrite or `pi --fork`. Three options were compared:

| Option | What it does | Cost |
|---|---|---|
| A. Copy verbatim | `cp src dst`; pi may complain about duplicate session ids | Cheapest; fragile |
| B. cp + bytes.ReplaceAll on session id | Read whole file, replace, write atomically | Medium; **chosen** |
| C. `pi --fork` non-interactively | Spawn pi twice; pi-canonical | Heaviest |

Why B is safe:
- Pi session ids are UUIDs (122 random bits). The chance the same byte sequence appears outside the session id field is effectively zero.
- `bytes.ReplaceAll` rewrites every occurrence consistently — covers `sessionId`, plus any extension entries that reference the session id.
- Message-level `id`/`parentId` are *different* UUIDs and stay valid; tree integrity preserved.

Algorithm:
```
1. newID = uuid v7 (lowercase, dashes)
2. oldID = read sessionId from src (first JSONL line, regex)
3. tmp = target + ".tmp"
4. cp src tmp
5. buf = read tmp; buf = bytes.ReplaceAll(buf, oldID, newID); write buf to tmp; fsync
6. rename(tmp, target)
7. on any error after step 4: os.Remove(tmp); restore quarantined file if applicable
```

No streaming line-by-line parse. No JSON re-encoding (preserves exact formatting; survives pi format additions). Atomic via single `rename` on POSIX.

Regex for oldID: `"sessionId"\s*:\s*"([0-9a-f-]{36})"`. If the first 64 KB don't match -> `SETY_IMPORT_FAILED reason=no_session_id_in_source`.

## --helpy output

**No filter, no search.** Print everything, exit. Filtering belongs in `grep`. Generated from the registry, not hand-typed.

### Text mode

```
cly pi — piwrap flags on top of the pi binary

NAMING
  -n, --name <name>
      Set session name. Renames cmux tab, exports $CLY_SESSION_NAME,
      pins a pi session file at:
        ~/.pi/agent/sessions/<encoded-cwd>/<prefix>-<kebab-name>.jsonl
      Prefix from modules.piwrap.session_file_name_prefix (default: cly).

      Examples:
        cly pi -n refactor-auth
        cly pi --name "My Work" -p "summarize"

SESSION IMPORT
  --sety session_import.id=<UUID|prefix|path>
      Fork an existing pi session into this cwd under -n's name.
      Requires -n. Source resolved from current cwd's session dir
      (or all dirs if modules.piwrap.session_import.search_scope=all).

  --sety session_import.override=true|false
      On filename conflict at the target, move existing file to
      modules.piwrap.session_import.quarantine_dir (default
      ~/.local/share/cly/trash/pi-sessions/) and proceed (true),
      or fail with SETY_IMPORT_CONFLICT (false).
      Default: modules.piwrap.session_import.override (false).

      Examples:
        cly pi -n bug-repro --sety session_import.id=019e5057
        cly pi -n bug-repro --sety session_import.id=$UUID \
            --sety session_import.override=true
        cly pi -n bug-repro --sety session_import.id=$UUID --dry-run

DRY RUN
  --dry-run
      Validate piwrap-side actions and print the planned operation as
      JSON. No filesystem writes, no pi exec.

ENV
  CLY_SESSION_NAME              Set automatically when -n is used.

See also:
  pi --help                     Passthrough flags (model, thinking, ...)
  cly --help                    Cobra command tree.
  cly pi --help                 This subcommand's stock help.
```

### JSON mode (`--helpy -o json`)

Same content as text, walked from the static registry, encoded with `encoding/json`. Doubles as the source for `cly schema cli` / `--describe`.

```json
{
  "version": "0.42.0",
  "sections": [
    {
      "id": "naming",
      "title": "Naming",
      "flags": [
        {
          "names": ["-n", "--name"],
          "value": "<name>",
          "description": "Set session name; renames cmux tab; pins pi session file.",
          "config_keys": ["modules.piwrap.session_file_name_prefix"],
          "env": ["CLY_SESSION_NAME"],
          "examples": [
            "cly pi -n refactor-auth",
            "cly pi --name \"My Work\" -p \"summarize\""
          ]
        }
      ]
    },
    {
      "id": "session-import",
      "title": "Session Import",
      "flags": [
        {
          "names": ["--sety session_import.id"],
          "value": "<UUID|prefix|path>",
          "description": "Fork an existing pi session into this cwd.",
          "requires": ["-n"],
          "config_keys": [
            "modules.piwrap.session_import.override",
            "modules.piwrap.session_import.search_scope",
            "modules.piwrap.session_import.quarantine_dir"
          ],
          "errors": [
            "SETY_IMPORT_NOT_FOUND",
            "SETY_IMPORT_AMBIGUOUS",
            "SETY_IMPORT_CONFLICT",
            "SETY_NAME_REQUIRED"
          ],
          "examples": [
            "cly pi -n bug-repro --sety session_import.id=019e5057",
            "cly pi -n bug-repro --sety session_import.id=$UUID --sety session_import.override=true"
          ]
        },
        {
          "names": ["--sety session_import.override"],
          "value": "true|false",
          "description": "On target conflict, move existing file to quarantine dir and proceed.",
          "config_keys": ["modules.piwrap.session_import.override"]
        }
      ]
    },
    {
      "id": "dry-run",
      "title": "Dry Run",
      "flags": [
        {
          "names": ["--dry-run"],
          "description": "Validate piwrap actions and print plan as JSON. No writes, no pi exec."
        }
      ]
    }
  ],
  "env": [
    {"name": "CLY_SESSION_NAME", "description": "Set automatically when -n is used."}
  ],
  "see_also": {
    "pi_help": "pi --help",
    "cobra_help": "cly --help",
    "piwrap_help": "cly pi --help"
  }
}
```

## Schema introspection (Principle 2)

```bash
cly pi schema sety
# {
#   "session_import.id":       {"type":"string","format":"uuid-or-prefix-or-path","minLength":8},
#   "session_import.override": {"type":"boolean","default":false,
#                               "configKey":"modules.piwrap.session_import.override"}
# }
```

## Dry-run (Principle 5)

```bash
cly pi -n my-session --sety session_import.id=$UUID --dry-run
# {
#   "action": "session.import",
#   "source": "/Users/.../019e5057-...jsonl",
#   "target": "/Users/.../cly-my-session.jsonl",
#   "conflict": true,
#   "override": false,
#   "would_quarantine": null,
#   "would_fork": false,
#   "blocked_by": "SETY_IMPORT_CONFLICT"
# }
```

No filesystem writes, no pi exec.

## Errors

JSON on stdout, hint on stderr, non-zero exit (Principle 6). All codes prefixed `SETY_*`.

| Code | When |
|---|---|
| `SETY_FORMAT` | `--sety` value isn't `key=value` |
| `SETY_UNKNOWN_KEY` | key not in {`session_import.id`, `session_import.override`} |
| `SETY_PARSE` | `override` not coercible to bool |
| `SETY_NAME_REQUIRED` | `session_import.id` set, `-n` missing |
| `SETY_IMPORT_ID_TOO_SHORT` | UUID prefix < 8 chars and not a path |
| `SETY_IMPORT_NOT_FOUND` | no session matches |
| `SETY_IMPORT_AMBIGUOUS` | >1 match; candidates listed on stderr |
| `SETY_IMPORT_CONFLICT` | target exists, override=false |
| `SETY_IMPORT_TARGET_BUSY` | target locked by another pi process |
| `SETY_IMPORT_FAILED` | copy/replace/rename failed (filesystem, missing sessionId, rename collision) |

Each carries `message`, `details.target`, `details.source`, `details.candidates` where relevant.

## Key Decisions

- `--sety` is **piwrap-only with fixed keys**, not a generic cly config setter. Two keys total: `session_import.id`, `session_import.override`. Anything else errors. Keeps the surface tiny and typed.
- Key namespace is `session_import.*` (underscore), not `session.import.*` (dot-nested). Single namespace, no hidden tree.
- Quarantine default: `~/.local/share/cly/trash/pi-sessions/` (sibling to existing `app.data_dir`). Files moved, never `rm`'d. Filename embeds UTC timestamp + encoded cwd + original basename so restoration is unambiguous.
- Fork strategy: `cp` + `bytes.ReplaceAll` of the top-level `sessionId`. No JSONL parsing. Atomic via `target.tmp` -> `rename`.
- `-n / --name` is required when `session_import.id` is set. Without it there's nowhere to land the fork.
- Search scope defaults to `cwd`. `all` is opt-in via config.
- `--helpy` has **no filter, no search**. Print everything, exit. Filtering belongs in `grep`.
- `--helpy` is generated from a registry, not a hand-written string. New flags must register or they're invisible — review-catchable.
- `--dry-run` returns JSON describing the planned operation. No writes, no pi exec.
- Errors: JSON on stdout, hint on stderr, non-zero exit. Codes prefixed `SETY_*`.
- Per-key flags like `--foo.a=bar` were considered and rejected: pflag doesn't accept dots in flag names without custom parsing, and the JSON payload form (Principle 1) is the better long-term path for agent invocation.

## Open Questions

- **Auto-show cheat sheet on first run?** Gated by a marker file under `~/.config/cly/`. Yes/no?
- **Unknown subcommand suggestion.** When `cly pi foo` doesn't match anything, suggest `cly pi --helpy` instead of cobra's default error?
- **Active-session lock detection.** Does pi write a lockfile sidecar we can check before stomping a target? If not, do we add `flock` ourselves or skip and rely on user discipline?
- **`schema sety` subcommand vs `--describe` flag.** Either works under `cly pi`. Pick one and stick to it.
- **Search scope when source is an absolute `.jsonl` path.** Confirmed irrelevant; use the path as-is, skip lookup.
- **Future second surface for agents.** Adopt `--json` payload (Principle 1) under `cly pi` as a parallel entry point routing to the same validator, or stay flag-only for now?
- **Resolved (no longer open):** `--helpy` naming collision with the `helpy` AI-chat subcommand. Not a problem — the cheat sheet flag lives on `cly pi`, while AI chat is `cly helpy`. Different command paths, no clash.

## Implementation Notes

- Order of work:
  1. `--sety` parser + key validation (`modules/piwrap/sety.go` + tests)
  2. Source resolver + quarantine + cp/replace fork (`modules/piwrap/import.go`, `fork.go` + tests)
  3. `--dry-run` JSON output
  4. `pkg/helpy` registry + text/JSON renderers
  5. Wire `--helpy` into `cmd/root.go`
  6. Register piwrap entries with helpy at package init
- Tests:
  - Unit: `kebabCase`, `encodeCwd` (already shipped), `parseSety`, `resolveSource`, `forkSession` (fixture JSONL -> assert sessionId rewritten + every other byte byte-equal).
  - Property: random valid UUIDs as oldID, assert ReplaceAll touches only those exact 36-byte windows.
  - Integration (skip if `pi` not on PATH): fork a real session, run `pi --session <forked> --no-tools -p ""`, assert exit 0.
- Use `pkg/style` for lipgloss in `--helpy` text renderer. Strip styling when `!isatty` or `NO_COLOR` is set.
- UUID v7 via `github.com/google/uuid` (verify go.mod before adding).
- `filepath.Separator` in `encodeCwd` for cross-platform behavior.
- Touchpoints in existing code: `modules/piwrap/piwrap.go` (extend `Run`), `modules/piwrap/cmd.go` (still `DisableFlagParsing: true`; no cobra flag declarations), `modules/config/config.yaml` (add defaults), new `pkg/helpy/` package. **Do not touch `cmd/root.go`** — these flags are scoped to `cly pi`.
- `--sety`, `--sety-string`, `--dry-run`, `--helpy` flag declaration: parsed by hand in piwrap because cobra flag parsing is disabled on the subcommand. The hand-parser strips them out of `args` before forwarding the rest to `pi`. Scan + remove pattern, similar to how `extractName` already works.
- Run `graphify update .` after the change because piwrap source moves.

## What we are explicitly NOT doing

- Generic cly config override via `--sety` (e.g. `--sety modules.piwrap.session_file_name_prefix=...`). Out of scope. `--sety` is typed and piwrap-scoped.
- Streaming line-by-line JSONL parse during fork. Whole-file read is fine for typical session sizes.
- Touching message-level `id`/`parentId`. Tree integrity matters; only the top-level session id is rewritten.
- Auto-deleting quarantined files. Manual or follow-up `cly piwrap gc` only.
- Filtering / searching `--helpy` output. Use `grep`.
- Inline `--json '{...}'` payload form. Possible future second surface; not in this draft.

