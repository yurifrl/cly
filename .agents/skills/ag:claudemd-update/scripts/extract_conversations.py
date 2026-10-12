#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""Extract Claude Code conversations - full dump."""

import argparse
import json
import os
import sys
from datetime import datetime, timedelta
from pathlib import Path


def get_project_key(project_path: str) -> str:
    key = project_path.replace("/", "-").replace("\\", "-")
    if not key.startswith("-"):
        key = "-" + key
    return key


def find_conversation_files(project_path: str, days: int) -> list[Path]:
    claude_dir = Path.home() / ".claude" / "projects"
    project_key = get_project_key(project_path)

    exact_match = claude_dir / project_key
    if exact_match.exists():
        project_dir = exact_match
    else:
        project_dirs = list(claude_dir.glob(f"*{project_key}*"))
        if not project_dirs:
            print(f"No conversations for: {project_path}", file=sys.stderr)
            return []
        project_dir = min(project_dirs, key=lambda p: len(str(p)))

    cutoff = datetime.now() - timedelta(days=days)
    files = []
    for f in project_dir.glob("*.jsonl"):
        mtime = datetime.fromtimestamp(f.stat().st_mtime)
        if mtime >= cutoff:
            files.append((f, mtime))

    files.sort(key=lambda x: x[1], reverse=True)
    return [f for f, _ in files]


def extract_messages(filepath: Path) -> list[str]:
    messages = []
    with open(filepath) as f:
        for line in f:
            try:
                entry = json.loads(line)
                if entry.get("type") == "user":
                    content = entry.get("message", {}).get("content")
                    if isinstance(content, str):
                        messages.append(content)
                    elif isinstance(content, list):
                        for item in content:
                            if isinstance(item, dict):
                                text = item.get("text", "")
                                if text and not text.startswith("<"):
                                    messages.append(text)
                            elif isinstance(item, str):
                                messages.append(item)
            except json.JSONDecodeError:
                continue
    return messages


def format_time_ago(dt: datetime) -> str:
    diff = datetime.now() - dt
    if diff.total_seconds() < 60:
        return "just now"
    elif diff.total_seconds() < 3600:
        return f"{int(diff.total_seconds() / 60)}m ago"
    elif diff.total_seconds() < 86400:
        return f"{int(diff.total_seconds() / 3600)}h ago"
    return f"{diff.days}d ago"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--project", default=os.getcwd())
    parser.add_argument("--days", type=int, default=1)
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    files = find_conversation_files(args.project, args.days)[:args.limit]

    results = []
    for filepath in files:
        mtime = datetime.fromtimestamp(filepath.stat().st_mtime)
        messages = extract_messages(filepath)

        clean = []
        for msg in messages:
            msg = msg.strip()
            if not msg or msg.startswith("[") or msg.startswith("{"):
                continue
            if msg.startswith("<local-command") or msg.startswith("<command-"):
                if "<command-name>" in msg:
                    start = msg.find("<command-name>") + 14
                    end = msg.find("</command-name>")
                    if end > start:
                        clean.append(f"[Command: {msg[start:end]}]")
                continue
            clean.append(msg)

        results.append({
            "id": filepath.stem,
            "time": mtime.isoformat(),
            "time_ago": format_time_ago(mtime),
            "messages": clean,
        })

    if args.json:
        print(json.dumps(results, indent=2))
    else:
        for i, conv in enumerate(results, 1):
            print(f"### {i}. [{conv['time_ago']}] {conv['id'][:8]}...")
            print(f"Messages: {len(conv['messages'])}")
            for msg in conv['messages']:
                print(f"  - {msg}")
            print()


if __name__ == "__main__":
    main()
