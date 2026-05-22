# piwrap session import + --helpy

Add a piwrap-scoped `--sety` flag that imports/forks an existing pi session into the current cwd under `-n`'s name, and a `--helpy` flag that prints cly's custom flags with examples.

> Note: This is a draft to organize ideas and scope before implementation.

## Goal

Make it trivial to take an existing pi session (by UUID or prefix) and fork it into the current cwd under a stable, named filename, with safe conflict handling. Surface cly's custom-on-top-of-pi flags through a single `--helpy` cheat sheet so users (and agents) don't have to grep the source.

## Architecture

`piwrap` parses `--sety key=value` (repeatable, fixed key set) and `-n name` before forwarding to `pi`. When `session_import.id` is set, piwrap runs an import pipeline (resolve source -> handle conflict -> cp + bytes.ReplaceAll the session id -> rename) then injects `--session <target>` into the args sent to pi. `--helpy` is a registry-driven cheat sheet rendered as text or JSON; it short-circuits before any subcommand runs.

## Components

- **`modules/piwrap/sety.go`** — parse `--sety key=value` (repeatable). Fixed keys: `session_import.id`, `session_import.override`. Bool coercion for override. Error codes: `SETY_FORMAT`, `SETY_UNKNOWN_KEY`, `SETY_PARSE`.
- **`modules/piwrap/import.go`** — resolve source (uuid / >=8-char prefix / absolute path), search scope (`cwd` | `all`), conflict handling, quarantine.
- **`modules/piwrap/fork.go`** — `cp src target.tmp` -> read oldSessionID via regex on first JSONL line -> `bytes.ReplaceAll` -> atomic rename. New UUID v7.
- **`modules/piwrap/piwrap.go`** — wire `--sety` parsing into `Run`, validate `-n` required when import.id present, call import pipeline before exec'ing pi, inject `--session <target>`.
- **`pkg/helpy/registry.go`** — typed `Entry` registry. Each piwrap flag registers itself at package init.
- **`pkg/helpy/render_text.go`** + **`render_json.go`** — text (lipgloss, NO_COLOR-aware) and JSON renderers.
- **`cmd/root.go`** — persistent `--helpy` flag; if set, render registry and `os.Exit(0)` before subcommand routing.
- **`modules/config/config.yaml`** — defaults for `modules.piwrap.session_import.{override, quarantine_dir, search_scope}`.

## Key Decisions

- `--sety` is **piwrap-only with fixed keys**, not a generic cly config setter. Two keys total: `session_import.id`, `session_import.override`. Anything else errors. Keeps the surface tiny and typed.
- Key namespace is `session_import.*` (underscore), not `session.import.*` (dot-nested). Single namespace, no hidden tree.
- Quarantine default: `~/.local/share/cly/trash/pi-sessions/` (sibling to existing `app.data_dir`). Overridden via `modules.piwrap.session_import.quarantine_dir`. Files moved, never `rm`'d. Filename embeds UTC timestamp + encoded cwd + original basename so restoration is unambiguous.
- Fork strategy: `cp` + `bytes.ReplaceAll` of the top-level `sessionId`. No JSONL parsing, no streaming rewrite. Safe because UUIDs don't collide with surrounding bytes. Atomic via `target.tmp` -> `rename`.
- `-n / --name` is required when `session_import.id` is set. Without it there's nowhere to land the fork.
- Search scope defaults to `cwd` (current cwd's encoded session dir only). `all` is opt-in.
- `--helpy` has **no filter, no search**. Print everything, exit. Filtering belongs in `grep`.
- `--helpy` is generated from a registry, not a hand-written string. New flags must register or they're invisible — review-catchable.
- `--dry-run` returns JSON describing the planned operation (source, target, conflict, would_quarantine, blocked_by). No writes, no pi exec.
- Errors follow Principle 6: JSON on stdout, hint on stderr, non-zero exit. Codes prefixed `SETY_*`.

## Open Questions

- **Flag name collision.** `--helpy` clashes with the existing `helpy` AI-chat subcommand. Pick: keep `--helpy`, rename to `--cheat`, or move to a `cly cheat` subcommand.
- **Auto-show cheat sheet on first run?** Gated by a marker file under `~/.config/cly/`. Yes/no?
- **Unknown subcommand suggestion.** When `cly foo` doesn't match any subcommand, suggest `cly --helpy` (or `cly cheat`) instead of cobra's default error?
- **Active-session lock detection.** Does pi write a lockfile sidecar we can check before stomping a target? If not, do we add `flock` ourselves or skip and rely on user discipline?
- **`schema sety` subcommand vs `--describe` flag.** Either works; pick one and stick to it across cly.
- **Search scope when source is an absolute `.jsonl` path.** Scope is irrelevant in that case. Confirm we just use the path as-is and skip the lookup.

## Implementation Notes

- Order of work: (1) `--sety` parser + key validation, (2) import resolver + quarantine + cp/replace, (3) `--dry-run` JSON, (4) `pkg/helpy` registry + renderers, (5) tests for each.
- Tests: unit for `kebabCase`/`encodeCwd` (already present), `parseSety`, `resolveSource`, `forkSession` (fixture JSONL -> assert sessionId rewritten + every other byte equal). Integration: real `pi --session <forked> -p ""` with `--no-tools`, gated on pi being on PATH.
- Use `pkg/style` for lipgloss in `--helpy` text renderer. Strip styling when `!isatty` or `NO_COLOR` is set.
- UUID v7 via `github.com/google/uuid` (already a transitive dep via go.mod — verify, don't add gratuitously).
- `filepath.Separator` everywhere in `encodeCwd` so behavior is correct cross-platform, even though pi's session layout is observed from macOS.
- Touchpoints in existing code: `modules/piwrap/piwrap.go` (extend `Run`), `modules/piwrap/cmd.go` (declare flags on the cobra command), `modules/config/config.yaml` (add defaults), `cmd/root.go` (wire `--helpy`).
- `--sety` flag declaration: `StringArrayVarP(&setyArgs, "sety", "y", nil, "...")`. Parsed in piwrap before any forwarding.
- Run `graphify update .` after the change because piwrap source moves.
