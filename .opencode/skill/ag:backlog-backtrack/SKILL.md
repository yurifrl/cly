---
name: ag:backlog-backtrack
description: Generate BACKLOG.md from Claude Code conversation history. Summarizes recent sessions grouped by project and date. Idempotent — safe to re-run.
---

# Backlog Backtrack

Scan `~/.claude/history.jsonl` and produce a `BACKLOG.md` summarizing what was worked on recently.

## Usage

```
/ag:backlog-backtrack              # Default: last 2 days, writes ./BACKLOG.md
/ag:backlog-backtrack --days 7     # Last 7 days
/ag:backlog-backtrack --dry-run    # Preview only
```

## Workflow

1. Run the script with user-specified flags (default: `--days 2`):

```bash
uv run home/.agents/skills/ag:backlog-backtrack/scripts/backlog_backtrack.py --dry-run --days 2
```

2. Present output to the user for review
3. If approved, run without `--dry-run` (or with `--output PATH` if user specifies)
4. If a `BACKLOG.md` already exists, the script reads its processed-sessions block and skips duplicates automatically

## Script Location

`home/.agents/skills/ag:backlog-backtrack/scripts/backlog_backtrack.py`

## Flags

- `--days N` — Window in days (default: 2)
- `--output PATH` — Output file path (default: `./BACKLOG.md`)
- `--dry-run` — Print to stdout, don't write file
- `--project PATH` — Filter to a single project path
