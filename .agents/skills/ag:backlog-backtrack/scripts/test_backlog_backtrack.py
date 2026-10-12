#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["pytest"]
# ///
"""Tests for backlog_backtrack.py — written first (TDD)."""

import json
import time
from datetime import datetime, timedelta
from pathlib import Path

import pytest

from backlog_backtrack import (
    clean_display,
    generate_markdown,
    group_by_project,
    parse_history,
    read_processed_ids,
)


def _write_history(path: Path, entries: list[dict]) -> None:
    with open(path, "w") as f:
        for entry in entries:
            f.write(json.dumps(entry) + "\n")


def _ms(dt: datetime) -> int:
    return int(dt.timestamp() * 1000)


class TestParseHistory:
    def test_filters_by_days(self, tmp_path):
        now = datetime.now()
        history = tmp_path / "history.jsonl"
        _write_history(history, [
            {"display": "recent work", "timestamp": _ms(now - timedelta(hours=1)),
             "project": "/home/user/projectA", "sessionId": "aaa-111"},
            {"display": "old work", "timestamp": _ms(now - timedelta(days=5)),
             "project": "/home/user/projectB", "sessionId": "bbb-222"},
        ])
        result = parse_history(history, days=2)
        assert "aaa-111" in result
        assert "bbb-222" not in result

    def test_dedupes_sessions(self, tmp_path):
        now = datetime.now()
        history = tmp_path / "history.jsonl"
        _write_history(history, [
            {"display": "first message", "timestamp": _ms(now - timedelta(hours=2)),
             "project": "/home/user/proj", "sessionId": "aaa-111"},
            {"display": "second message", "timestamp": _ms(now - timedelta(hours=1)),
             "project": "/home/user/proj", "sessionId": "aaa-111"},
        ])
        result = parse_history(history, days=2)
        assert len(result) == 1
        assert "aaa-111" in result

    def test_uses_first_non_empty_display(self, tmp_path):
        now = datetime.now()
        history = tmp_path / "history.jsonl"
        _write_history(history, [
            {"display": "", "timestamp": _ms(now - timedelta(hours=3)),
             "project": "/home/user/proj", "sessionId": "aaa-111"},
            {"display": "the real summary", "timestamp": _ms(now - timedelta(hours=2)),
             "project": "/home/user/proj", "sessionId": "aaa-111"},
            {"display": "later message", "timestamp": _ms(now - timedelta(hours=1)),
             "project": "/home/user/proj", "sessionId": "aaa-111"},
        ])
        result = parse_history(history, days=2)
        assert result["aaa-111"]["display"] == "the real summary"

    def test_uses_latest_timestamp(self, tmp_path):
        now = datetime.now()
        history = tmp_path / "history.jsonl"
        ts_early = _ms(now - timedelta(hours=3))
        ts_late = _ms(now - timedelta(hours=1))
        _write_history(history, [
            {"display": "first", "timestamp": ts_early,
             "project": "/home/user/proj", "sessionId": "aaa-111"},
            {"display": "second", "timestamp": ts_late,
             "project": "/home/user/proj", "sessionId": "aaa-111"},
        ])
        result = parse_history(history, days=2)
        assert result["aaa-111"]["timestamp"] == ts_late


class TestCleanDisplay:
    def test_strips_commands(self):
        assert clean_display("/commit") is None
        assert clean_display("/save") is None
        assert clean_display("/help") is None

    def test_strips_pasted_text(self):
        assert clean_display("[Pasted text #1 +27 lines]") is None

    def test_strips_pasted_text_prefix(self):
        result = clean_display("[Pasted text #1 +84 lines]\n\nwhats going on?")
        assert result == "whats going on?"

    def test_strips_single_word(self):
        assert clean_display("continue") is None
        assert clean_display("yes") is None
        assert clean_display("ok") is None

    def test_truncates_long_text(self):
        long_text = "a " * 200
        result = clean_display(long_text)
        assert result is not None
        assert len(result) <= 123  # 120 + "..."

    def test_keeps_meaningful_text(self):
        assert clean_display("Add field to table with user") == "Add field to table with user"

    def test_strips_empty(self):
        assert clean_display("") is None
        assert clean_display("   ") is None


class TestGroupByProject:
    def test_groups_correctly(self):
        now = datetime.now()
        sessions = {
            "aaa": {"project": "/home/user/projectA", "display": "task A",
                    "timestamp": _ms(now)},
            "bbb": {"project": "/home/user/projectB", "display": "task B",
                    "timestamp": _ms(now)},
            "ccc": {"project": "/home/user/projectA", "display": "task A2",
                    "timestamp": _ms(now - timedelta(hours=1))},
        }
        grouped = group_by_project(sessions)
        assert "projectA" in grouped
        assert "projectB" in grouped
        assert len(grouped["projectA"]) == 2
        assert len(grouped["projectB"]) == 1

    def test_sorted_by_project_name(self):
        now = datetime.now()
        sessions = {
            "a": {"project": "/z-proj", "display": "z", "timestamp": _ms(now)},
            "b": {"project": "/a-proj", "display": "a", "timestamp": _ms(now)},
        }
        grouped = group_by_project(sessions)
        keys = list(grouped.keys())
        assert keys == ["a-proj", "z-proj"]

    def test_sessions_sorted_by_timestamp_desc(self):
        now = datetime.now()
        sessions = {
            "old": {"project": "/proj", "display": "old",
                    "timestamp": _ms(now - timedelta(hours=2))},
            "new": {"project": "/proj", "display": "new",
                    "timestamp": _ms(now)},
        }
        grouped = group_by_project(sessions)
        assert grouped["proj"][0]["session_id"] == "new"
        assert grouped["proj"][1]["session_id"] == "old"


class TestReadProcessedIds:
    def test_parses_comment_block(self, tmp_path):
        backlog = tmp_path / "BACKLOG.md"
        backlog.write_text(
            "# Backlog\n\nSome content\n\n"
            "<!-- processed-sessions\n"
            "aaa-111\n"
            "bbb-222\n"
            "-->\n"
        )
        ids = read_processed_ids(backlog)
        assert ids == {"aaa-111", "bbb-222"}

    def test_empty_when_no_block(self, tmp_path):
        backlog = tmp_path / "BACKLOG.md"
        backlog.write_text("# Backlog\n\nSome content\n")
        ids = read_processed_ids(backlog)
        assert ids == set()

    def test_empty_when_file_missing(self, tmp_path):
        backlog = tmp_path / "BACKLOG.md"
        ids = read_processed_ids(backlog)
        assert ids == set()


class TestGenerateMarkdown:
    def test_new_generation(self):
        now = datetime.now()
        ts = _ms(now)
        grouped = {
            "myproject": [
                {"session_id": "abc12345-full-id", "display": "Add feature X",
                 "timestamp": ts},
            ]
        }
        md, new_ids = generate_markdown(grouped, set())
        assert "# Backlog" in md
        assert "## myproject" in md
        assert "Add feature X [abc12345]" in md
        assert "abc12345-full-id" in md  # in processed block
        assert "abc12345-full-id" in new_ids

    def test_dedup_skips_processed(self):
        now = datetime.now()
        ts = _ms(now)
        grouped = {
            "proj": [
                {"session_id": "already-done-id", "display": "Old task",
                 "timestamp": ts},
                {"session_id": "new-task-id", "display": "New task",
                 "timestamp": ts},
            ]
        }
        md, new_ids = generate_markdown(grouped, {"already-done-id"})
        assert "Old task" not in md
        assert "New task [new-task" in md
        assert "already-done-id" in md  # still in processed block
        assert "new-task-id" in md

    def test_idempotent(self):
        now = datetime.now()
        ts = _ms(now)
        grouped = {
            "proj": [
                {"session_id": "aaa-111", "display": "Task one", "timestamp": ts},
            ]
        }
        md1, ids1 = generate_markdown(grouped, set())
        md2, ids2 = generate_markdown(grouped, ids1)
        # Second run should have no visible entries (all processed)
        # but processed block should be identical
        assert "Task one" not in md2
        assert "aaa-111" in md2  # still tracked

    def test_groups_by_date(self):
        now = datetime.now()
        yesterday = now - timedelta(days=1)
        grouped = {
            "proj": [
                {"session_id": "aaa", "display": "Today task",
                 "timestamp": _ms(now)},
                {"session_id": "bbb", "display": "Yesterday task",
                 "timestamp": _ms(yesterday)},
            ]
        }
        md, _ = generate_markdown(grouped, set())
        today_str = now.strftime("%Y-%m-%d")
        yesterday_str = yesterday.strftime("%Y-%m-%d")
        assert f"### {today_str}" in md
        assert f"### {yesterday_str}" in md


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
