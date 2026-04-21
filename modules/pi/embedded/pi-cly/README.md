# pi-cly

Pi extension shipped by cly.

## Commands

### `/save [name] [description="..."]`

Drives `cly as save` (the `as` alias of `agent-session`).

Prefilled values (computed in this extension, deterministic):
- `id` — read from the first JSON line of the session file (`ctx.sessionManager.getSessionFile()` → `obj.id`), same as checkpoint.ts does. Falls back to `<cwd-slug>-<timestamp>` only if unavailable.
- `name` — `pi.getSessionName()` (the auto-generated session summary from pi's session-summary extension); falls back to `<cwd-slug>`.
- `description` — `pi session in <cwd>`

Override forms:

| Input                                                  | Effective name     | Effective description |
|--------------------------------------------------------|--------------------|------------------------|
| `/save`                                                | prefill            | prefill                |
| `/save refactor notes`                                 | `refactor notes`   | prefill                |
| `/save description="auth bug notes"`                   | prefill            | `auth bug notes`       |
| `/save refactor notes description="auth bug notes"`    | `refactor notes`   | `auth bug notes`       |

Only `description` is currently accepted as a kv override. Unknown keys are ignored.

## Install (via cly)

```
cly pi extensions install
```

This copies this directory into `~/.pi/agent/extensions/pi-cly/`.

## Manual test plan

1. Build cly: `task build` (or `go install .`).
2. Install the extension: `cly pi extensions install`.
3. Confirm `~/.pi/agent/extensions/pi-cly/` contains `package.json` and `index.ts`.
4. Start pi in a repo; invoke `/save`. Expect `cly as save ...` to run.
5. Invoke `/save my session`. Expect `name` to be `my session`.
6. Invoke `/save my session description="what I did"`. Expect both overrides honored.
7. In all cases, pi should surface stdout/stderr from the cly call.
