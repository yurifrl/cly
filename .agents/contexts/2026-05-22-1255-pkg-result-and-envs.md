---
created: 2026-05-22T15:55:00Z
project: cly
description: piwrap pre-creates a named pi session file when --name is passed
context: piwrap session naming, pi --session resolution, cly --name workflow
tags: [piwrap, pi, sessions, cmux]
session_name: 2026-05-22-1255-pkg-result-and-envs
purpose: Make `cly --name foo` create/reuse a stable pi session file per cwd, mirroring pi's session directory layout.
session_id: 019e5057-19ae-7ddc-a9e2-42abd19c8053
provider: pi
resume_with: cly agent-session resume --provider pi 2026-05-22-1255-pkg-result-and-envs
context_name: 2026-05-22-1255-pkg-result-and-envs
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-05-22-1255-pkg-result-and-envs.md
---

# Session: 2026-05-22-1255-pkg-result-and-envs

- Name: 2026-05-22-1255-pkg-result-and-envs
- Purpose: Extend `piwrap` so `--name`/`-n` also pre-creates and pins a named pi session file under the cwd-encoded session dir.
- Resume: `cly agent-session resume --provider pi 2026-05-22-1255-pkg-result-and-envs`

## Context

`modules/piwrap` already intercepted `--name`/`-n`, set `$CLY_SESSION_NAME`, and renamed the cmux tab. The user wanted the same name to drive pi's session file path so re-invoking `cly -n foo` in the same cwd reopens the same conversation.

We confirmed via `pi --help` and `docs/sessions.md` that:
- `--session <path|id>` either resolves an existing session by partial UUID OR uses a literal path. Bare unknown UUIDs fail with `No session found matching '<id>'`.
- Anything containing `/`, `.`, or `\` is treated as a path; pi creates the file if it doesn't exist.
- pi's per-cwd dir is `~/.pi/agent/sessions/--<cwd-encoded>--/` where encoding is: strip leading `/`, replace `/` with `-`, wrap with `--...--`. Verified against `--Users-yuri-Workdir-Yuri-cly--` and the nested `--Users-yuri-.pi-agent-sessions---Users-yuri-Workdir-Nsx-...---certification----` example.

## Problem

Bare UUIDs passed to `--session` are treated as lookup keys, not creation hints, so we can't pre-mint a session by ID. Solution: pass a full `.jsonl` path. piwrap should compute that path from `--name` automatically, without overriding a user-supplied `--session`.

## Decisions

- Filename pattern: `cly-<kebab(name)>.jsonl` (prefix makes cly-managed sessions easy to grep in `pi -r`).
- Directory: `~/.pi/agent/sessions/<encodeCwd(cwd)>/` so `pi -c`/`pi -r` from the same cwd discovers it naturally.
- Skip injection if the user already passed `--session` or `--session=...` (respect explicit override). `--session-dir` is intentionally ignored — if you set it, pass `--session` too.
- Session ID inside the JSONL is still pi-generated; the filename is the stable handle. Documented as a caveat to the user.
- `kebabCase` is custom (lowercase ASCII, runs of non-alnum collapse to single dash, trim) — no new deps.
- `encodeCwd` uses `filepath.Separator` so behavior stays correct cross-platform, even though the layout is observed from macOS.

## Current State

Done. `modules/piwrap/piwrap.go` now injects `--session <path>` after `--name`/`-n`, and `modules/piwrap/piwrap_test.go` covers `kebabCase`, `encodeCwd`, `buildSessionPath`, and `hasSessionFlag`. `go test ./modules/piwrap/...` and `go build ./...` both green. Not committed.

## Next Steps

- (optional) Detect `--session-dir` and rebase the injected path on it for users who relocate sessions.
- (optional) Surface the resolved session path on stderr when injected, so users see where the file landed.
- (optional) Add an integration test that boots `pi --version` via piwrap to verify the args round-trip without breaking.
- Run `graphify update .` since piwrap source changed.
