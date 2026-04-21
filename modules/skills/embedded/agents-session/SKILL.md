---
name: agents-session
description: Save and resume AI agent sessions via cly. Use when starting, checkpointing, or ending a coding agent session, or when the user mentions saving the current session, naming it, or resuming prior work. Drives `cly agent-session save` (alias `cly as save`) with best-effort id, name, and description derived from conversation context.
---

# agents-session

Guidance for AI agents running a coding session backed by cly's agent-session store.

## When to use

- The user asks to "save the session", "checkpoint this", "remember this work", or similar.
- You are about to close out work worth resuming later (end of a task, end of a branch, end of the day).
- You are starting fresh and want to attach to a named session.

## The command

cly ships an agent-session store. The save (upsert) command is:

```
cly agent-session save <id> [name] [description]
# alias: cly as save
```

Positional args: `id` (required), `name` (optional), `description` (optional).
Flags: `-n/--name`, `-d/--description`, `--set key=value`, `--meta '{...}'`, `--override`.

The command is an upsert: it creates the entry if missing, updates it if present.

## How to fill the fields (best effort)

You will not always have perfect information. Extract what you can from the conversation and the repo:

- **id**
  - Prefer a stable slug derived from the current branch, task, or PR (e.g. `auth-refactor-2026-04`).
  - If nothing obvious, use a short lowercase kebab-case slug of the primary topic.
  - Do not invent UUIDs unless the user asks for one.

- **name**
  - A short human-readable title for the session (e.g. `"Auth refactor: split provider layer"`).
  - If the user gave one inline (`/save some name here`), use it verbatim.

- **description**
  - A one-to-three sentence summary of what was done and what is next.
  - Prefer concrete signal (files touched, open questions, next action) over generic prose.
  - If the user passed `description="..."`, use it verbatim.

When in doubt, ask the user once for a better value rather than guessing a poor one.

## Examples

Minimal:
```
cly as save auth-refactor
```

With name:
```
cly as save auth-refactor "Auth refactor: split provider layer"
```

Full:
```
cly as save auth-refactor \
  "Auth refactor: split provider layer" \
  "Extracted providers.go, tests green. Next: migrate tui.go to new interface."
```

## Notes

- This skill is best-effort by design. The pi `/save` extension provides a deterministic path for interactive use.
- If `cly` is not on PATH, say so instead of falling back to a different tool.
- Do not touch production or send any message as part of "saving" — this is local state only.
