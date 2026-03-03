#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Read Claude Code history and generate BACKLOG.md summarizing recent work."""

import argparse
import json
import re
import sys
from datetime import datetime, timedelta
from pathlib import Path


def parse_history(path: Path, days: int) -> dict[str, dict]:
    """Read history.jsonl, filter by days window, dedupe by sessionId.

    Returns {sessionId: {project, display, timestamp}}.
    Uses first non-empty display per session, latest timestamp.
    """
    cutoff = datetime.now() - timedelta(days=days)
    cutoff_ms = int(cutoff.timestamp() * 1000)

    sessions: dict[str, dict] = {}
    with open(path) as f:
        for line in f:
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue

            ts = entry.get("timestamp", 0)
            if ts < cutoff_ms:
                continue

            sid = entry.get("sessionId", "")
            if not sid:
                continue

            if sid not in sessions:
                sessions[sid] = {
                    "project": entry.get("project", ""),
                    "display": "",
                    "timestamp": ts,
                }

            # Use first non-empty display
            if not sessions[sid]["display"]:
                display = entry.get("display", "")
                cleaned = clean_display(display)
                if cleaned:
                    sessions[sid]["display"] = cleaned

            # Track latest timestamp
            if ts > sessions[sid]["timestamp"]:
                sessions[sid]["timestamp"] = ts

    return sessions


def clean_display(text: str) -> str | None:
    """Strip noise from display text. Returns None if not meaningful."""
    text = text.strip()
    if not text:
        return None

    # Strip slash commands
    if text.startswith("/"):
        return None

    # Strip pasted text markers (standalone or as prefix)
    text = re.sub(r"^\[Pasted text[^\]]*\]\s*", "", text).strip()
    if not text:
        return None

    # Strip single-word low-signal entries
    if len(text.split()) <= 1:
        return None

    # Truncate long text
    if len(text) > 120:
        text = text[:120] + "..."

    return text


def group_by_project(sessions: dict[str, dict]) -> dict[str, list[dict]]:
    """Group sessions by project basename, sorted by project name.

    Each session list sorted by timestamp descending.
    """
    groups: dict[str, list[dict]] = {}
    for sid, info in sessions.items():
        # Skip sessions with no meaningful display
        if not info["display"]:
            continue
        project_name = Path(info["project"]).name if info["project"] else "unknown"
        if project_name not in groups:
            groups[project_name] = []
        groups[project_name].append({
            "session_id": sid,
            "display": info["display"],
            "timestamp": info["timestamp"],
        })

    # Sort groups by project name
    sorted_groups = dict(sorted(groups.items()))

    # Sort sessions within each group by timestamp desc
    for sessions_list in sorted_groups.values():
        sessions_list.sort(key=lambda s: s["timestamp"], reverse=True)

    return sorted_groups


def read_processed_ids(backlog_path: Path) -> set[str]:
    """Parse <!-- processed-sessions ... --> comment from existing BACKLOG.md."""
    if not backlog_path.exists():
        return set()

    content = backlog_path.read_text()
    match = re.search(
        r"<!-- processed-sessions\n(.*?)-->",
        content,
        re.DOTALL,
    )
    if not match:
        return set()

    ids = set()
    for line in match.group(1).strip().splitlines():
        line = line.strip()
        if line:
            ids.add(line)
    return ids


def generate_markdown(
    grouped: dict[str, list[dict]],
    processed_ids: set[str],
) -> tuple[str, set[str]]:
    """Generate BACKLOG.md content.

    Returns (markdown_string, all_processed_ids).
    Skips sessions already in processed_ids.
    """
    all_ids = set(processed_ids)
    lines = ["# Backlog", ""]

    has_content = False
    for project_name, sessions in grouped.items():
        # Filter out already-processed sessions
        new_sessions = [s for s in sessions if s["session_id"] not in processed_ids]
        if not new_sessions:
            # Still collect IDs
            for s in sessions:
                all_ids.add(s["session_id"])
            continue

        has_content = True
        lines.append(f"## {project_name}")
        lines.append("")

        # Group by date
        by_date: dict[str, list[dict]] = {}
        for s in new_sessions:
            all_ids.add(s["session_id"])
            dt = datetime.fromtimestamp(s["timestamp"] / 1000)
            date_str = dt.strftime("%Y-%m-%d")
            if date_str not in by_date:
                by_date[date_str] = []
            by_date[date_str].append(s)

        # Dates sorted descending (newest first)
        for date_str in sorted(by_date.keys(), reverse=True):
            lines.append(f"### {date_str}")
            for s in by_date[date_str]:
                short_id = s["session_id"][:8]
                lines.append(f"- {s['display']} [{short_id}]")
            lines.append("")

    if not has_content:
        lines.append("*No new sessions to report.*")
        lines.append("")

    # Append processed-sessions comment block
    lines.append("<!-- processed-sessions")
    for sid in sorted(all_ids):
        lines.append(sid)
    lines.append("-->")
    lines.append("")

    return "\n".join(lines), all_ids


def main():
    parser = argparse.ArgumentParser(
        description="Generate BACKLOG.md from Claude Code conversation history",
    )
    parser.add_argument("--days", type=int, default=2, help="Window in days (default: 2)")
    parser.add_argument("--output", type=str, default="./BACKLOG.md", help="Output path")
    parser.add_argument("--dry-run", action="store_true", help="Print to stdout, don't write")
    parser.add_argument("--project", type=str, help="Filter to single project path")
    args = parser.parse_args()

    history_path = Path.home() / ".claude" / "history.jsonl"
    if not history_path.exists():
        print("No history.jsonl found at ~/.claude/history.jsonl", file=sys.stderr)
        sys.exit(1)

    sessions = parse_history(history_path, args.days)

    # Filter by project if specified
    if args.project:
        sessions = {
            sid: info for sid, info in sessions.items()
            if info["project"] == args.project
        }

    grouped = group_by_project(sessions)
    output_path = Path(args.output)
    processed_ids = read_processed_ids(output_path)
    md, _ = generate_markdown(grouped, processed_ids)

    if args.dry_run:
        print(md)
    else:
        output_path.write_text(md)
        print(f"Wrote {output_path}", file=sys.stderr)


if __name__ == "__main__":
    main()
