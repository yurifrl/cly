package ompsummary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFoldTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanCompactions(t *testing.T) {
	foldAt := float64(time.Date(2026, 9, 19, 7, 39, 12, 0, time.UTC).Unix())

	t.Run("semantic handoff distills its Goal line", func(t *testing.T) {
		p := writeFoldTranscript(t,
			`{"type":"message","id":"m1","timestamp":"2026-09-19T07:00:00.000Z","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}`,
			`{"type":"compaction","id":"c1","timestamp":"2026-09-19T07:39:12.145Z","summary":"## Goal\n\nFix the **broken** link\n\n## Progress\n\ndone"}`,
		)
		fold := ScanCompactions(p)
		if fold.Count != 1 {
			t.Fatalf("count = %d, want 1", fold.Count)
		}
		if fold.At != foldAt {
			t.Errorf("lastAt = %v, want %v", fold.At, foldAt)
		}
		if fold.Note != "Fix the broken link" {
			t.Errorf("note = %q, want %q", fold.Note, "Fix the broken link")
		}
		if fold.Summary == "" {
			t.Error("Summary empty, want raw fold summary")
		}
	})

	t.Run("mechanical fold distills dropped-history size", func(t *testing.T) {
		p := writeFoldTranscript(t,
			`{"type":"compaction","id":"c2","timestamp":"2026-09-19T08:00:00.000Z","summary":"Resume prior conversation.\nReading HISTORY:\n- About 326829 characters of older middle history dropped to fit archive budget.\nHISTORY"}`,
		)
		fold := ScanCompactions(p)
		if fold.Count != 1 {
			t.Fatalf("count = %d, want 1", fold.Count)
		}
		if fold.Note != "~327k older chars dropped" {
			t.Errorf("note = %q, want %q", fold.Note, "~327k older chars dropped")
		}
	})

	t.Run("last record supplies the note and timestamp", func(t *testing.T) {
		p := writeFoldTranscript(t,
			`{"type":"compaction","id":"c1","timestamp":"2026-09-19T07:39:12.145Z","summary":"## Goal\n\nfirst fold"}`,
			`{"type":"compaction","id":"c2","timestamp":"2026-09-19T08:00:00.000Z","summary":"- About 5000 characters of older middle history dropped to fit archive budget."}`,
		)
		fold := ScanCompactions(p)
		if fold.Count != 2 {
			t.Fatalf("count = %d, want 2", fold.Count)
		}
		wantAt := float64(time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC).Unix())
		if fold.At != wantAt {
			t.Errorf("lastAt = %v, want %v", fold.At, wantAt)
		}
		if fold.Note != "~5k older chars dropped" {
			t.Errorf("note = %q, want %q", fold.Note, "~5k older chars dropped")
		}
	})

	t.Run("nested compaction mention is not a fold", func(t *testing.T) {
		p := writeFoldTranscript(t,
			`{"type":"message","id":"m1","inner":{"type":"compaction"},"message":{"role":"assistant","content":[]}}`,
		)
		fold := ScanCompactions(p)
		if fold.Count != 0 || fold.At != 0 || fold.Note != "" {
			t.Fatalf("count=%d at=%v note=%q, want all zero", fold.Count, fold.At, fold.Note)
		}
	})

	t.Run("broken lines are skipped", func(t *testing.T) {
		p := writeFoldTranscript(t,
			`{"type":"compaction","id":"c1",TORN`,
			`{"type":"compaction","id":"c2","timestamp":"2026-09-19T07:39:12.145Z","summary":"## Goal\n\nok"}`,
		)
		fold := ScanCompactions(p)
		if fold.Count != 1 || fold.Note != "ok" {
			t.Fatalf("count=%d note=%q, want 1/ok", fold.Count, fold.Note)
		}
	})

	t.Run("missing transcript yields zero", func(t *testing.T) {
		fold := ScanCompactions(filepath.Join(t.TempDir(), "absent.jsonl"))
		if fold.Count != 0 || fold.At != 0 || fold.Note != "" {
			t.Fatalf("count=%d at=%v note=%q, want all zero", fold.Count, fold.At, fold.Note)
		}
	})
}
