package ompsummary

import (
	"context"
	"strings"
	"testing"
	"time"
)

func testSummarizer() *Summarizer {
	// Constructed directly (not NewSummarizer) to stay hermetic: no global
	// config, no API-key probe, no provider resolution.
	return &Summarizer{
		cfg:     Config{Debounce: 30 * time.Second},
		running: map[string]context.CancelFunc{},
		HasKey:  true,
	}
}

func TestEnqueuePromotesFocusedJob(t *testing.T) {
	s := testSummarizer()
	tgt := Target{SessionID: "s1", CWD: t.TempDir()}
	s.Enqueue(Job{Target: tgt, Fingerprint: "fp1"})
	s.Enqueue(Job{Target: tgt, Fingerprint: "fp2", Focused: true})

	if len(s.queue) != 1 {
		t.Fatalf("queue = %d entries, want 1 (same session deduped)", len(s.queue))
	}
	j, ok := s.pending()
	if !ok || j.Target.SessionID != "s1" || !j.Focused || j.Fingerprint != "fp2" {
		t.Fatalf("pending = %+v ok=%v, want promoted focused job with fp2", j, ok)
	}
	if _, ok := s.pending(); ok {
		t.Fatal("queue should be empty after pop")
	}
}

func TestPendingFocusedJumpsAhead(t *testing.T) {
	s := testSummarizer()
	tmp := t.TempDir()
	s.Enqueue(Job{Target: Target{SessionID: "bg", CWD: tmp}, Fingerprint: "1"})
	s.Enqueue(Job{Target: Target{SessionID: "fg", CWD: tmp}, Fingerprint: "2", Focused: true})

	j, _ := s.pending()
	if j.Target.SessionID != "fg" {
		t.Fatalf("pending = %s, want fg (focused first)", j.Target.SessionID)
	}
	j, _ = s.pending()
	if j.Target.SessionID != "bg" {
		t.Fatalf("pending = %s, want bg (FIFO remainder)", j.Target.SessionID)
	}
}

func TestEnqueueNoKeyIsNoop(t *testing.T) {
	s := testSummarizer()
	s.HasKey = false
	s.Enqueue(Job{Target: Target{SessionID: "s1", CWD: t.TempDir()}})
	if len(s.queue) != 0 {
		t.Fatalf("queue = %d, want 0 without a key", len(s.queue))
	}
}

func TestDrainDropsStaleFingerprints(t *testing.T) {
	s := testSummarizer()
	tmp := t.TempDir()
	s.Enqueue(Job{Target: Target{SessionID: "fresh", CWD: tmp}, Fingerprint: "same"})
	s.Enqueue(Job{Target: Target{SessionID: "stale", CWD: tmp}, Fingerprint: "old"})
	s.Drain(map[string]string{"fresh": "same", "stale": "changed"})

	if len(s.queue) != 1 {
		t.Fatalf("queue = %d, want 1", len(s.queue))
	}
	j, _ := s.pending()
	if j.Target.SessionID != "fresh" {
		t.Fatalf("kept %s, want fresh", j.Target.SessionID)
	}
}

func TestShouldSummarize(t *testing.T) {
	cwd := t.TempDir()
	s := testSummarizer()
	tgt := Target{SessionID: "s1", CWD: cwd}
	now := nowUnix()

	if !s.ShouldSummarize(tgt, "fp1", now) {
		t.Fatal("no state yet → should summarize")
	}

	save := func(fp, status string) {
		st := LoadState(cwd)
		st.Update("s1", &Summary{Status: status, Fingerprint: fp, UpdatedAt: now})
		if err := SaveState(cwd, st); err != nil {
			t.Fatal(err)
		}
	}

	save("fp1", "ok")
	if s.ShouldSummarize(tgt, "fp1", now) {
		t.Fatal("ok with same fingerprint → skip")
	}
	if !s.ShouldSummarize(tgt, "fp2", now) {
		t.Fatal("fingerprint changed → re-summarize")
	}

	save("fp1", "error")
	if s.ShouldSummarize(tgt, "fp1", now) {
		t.Fatal("recent error within debounce → back off")
	}
	if !s.ShouldSummarize(tgt, "fp1", now+61) {
		t.Fatal("error past debounce → retry")
	}

	save("fp1", "running")
	if s.ShouldSummarize(tgt, "fp1", now) {
		t.Fatal("running with same fingerprint → skip")
	}
	focused := tgt
	focused.Focused = true
	if !s.ShouldSummarize(focused, "fp2", now) {
		t.Fatal("running but stale fingerprint + focused → re-run")
	}
}

func TestNormalizeSquashesAndCaps(t *testing.T) {
	in := "first   paragraph.\n\n\n\tsecond  paragraph."
	out := normalize(in)
	if out != "first paragraph. second paragraph." {
		t.Fatalf("normalize = %q", out)
	}

	long := strings.Repeat("word ", 1000)
	if got := len([]rune(normalize(long))); got > 200 {
		t.Fatalf("normalize len = %d, want hard cap at 200 runes", got)
	}
}
