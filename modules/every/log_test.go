package every

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAppendAndReadLog(t *testing.T) {
	dir := t.TempDir()
	p := LogPath(dir, "demo")
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	if err := AppendLog(p, Event{TS: now, Event: "start", Extra: map[string]any{"run": 1}}); err != nil {
		t.Fatal(err)
	}
	if err := AppendLog(p, Event{TS: now.Add(time.Second), Event: "end", Extra: map[string]any{"run": 1, "exit": 0, "duration_ms": int64(1000)}}); err != nil {
		t.Fatal(err)
	}
	events, err := ReadLog(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].Event != "start" || events[1].Event != "end" {
		t.Fatalf("unexpected order: %+v", events)
	}
}

func TestReadLogSkipsBadLines(t *testing.T) {
	dir := t.TempDir()
	p := LogPath(dir, "demo")
	body := "{\"ts\":\"2026-05-25T12:00:00Z\",\"event\":\"start\"}\nnot-json\n{\"ts\":\"2026-05-25T12:00:01Z\",\"event\":\"end\"}\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	events, err := ReadLog(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 valid events, got %d", len(events))
	}
}

func TestMaybeTrimLogSize(t *testing.T) {
	dir := t.TempDir()
	p := LogPath(dir, "big")
	// write enough events to push past 1MB
	now := time.Now()
	var buf bytes.Buffer
	for i := 0; i < 30000; i++ {
		ev := Event{TS: now, Event: "spam", Extra: map[string]any{"i": i, "pad": strings.Repeat("x", 40)}}
		data, _ := json.Marshal(ev)
		buf.Write(data)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MaybeTrimLog(p, now, false); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Size() > MaxLogBytes {
		t.Fatalf("log not trimmed: %d bytes", info.Size())
	}
}

func TestMaybeTrimLogRetention(t *testing.T) {
	dir := t.TempDir()
	p := LogPath(dir, "old")
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	if err := AppendLog(p, Event{TS: old, Event: "start"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendLog(p, Event{TS: now, Event: "end"}); err != nil {
		t.Fatal(err)
	}
	if err := MaybeTrimLog(p, now, true); err != nil {
		t.Fatal(err)
	}
	events, _ := ReadLog(p)
	if len(events) != 1 || events[0].Event != "end" {
		t.Fatalf("retention drop failed: %+v", events)
	}
}

func TestLastNAndFilter(t *testing.T) {
	now := time.Now()
	events := []Event{
		{TS: now.Add(-3 * time.Hour), Event: "a"},
		{TS: now.Add(-1 * time.Hour), Event: "b"},
		{TS: now, Event: "c"},
	}
	got := LastN(events, 2)
	if len(got) != 2 || got[0].Event != "b" || got[1].Event != "c" {
		t.Fatalf("LastN: %+v", got)
	}
	since := FilterSince(events, now.Add(-2*time.Hour))
	if len(since) != 2 {
		t.Fatalf("FilterSince: %+v", since)
	}
}
