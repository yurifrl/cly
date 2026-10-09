package ompsummary

import (
	"context"
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

func TestEnqueueBusyDropsWithoutCancelling(t *testing.T) {
	s := testSummarizer()
	tmp := t.TempDir()
	cancelled := 0
	s.running["live"] = func() { cancelled++ }
	s.Enqueue(Job{Target: Target{SessionID: "live", CWD: tmp}, Fingerprint: "fp1", Focused: true})

	if len(s.queue) != 0 {
		t.Fatalf("queue = %d, want 0 while a run is in flight", len(s.queue))
	}
	if cancelled != 0 {
		t.Fatalf("cancelled = %d, want 0 (in-flight run must finish and save)", cancelled)
	}
}

func TestPendingReservesRunningSlot(t *testing.T) {
	s := testSummarizer()
	s.Enqueue(Job{Target: Target{SessionID: "s1", CWD: t.TempDir()}, Fingerprint: "fp1"})
	if s.Idle() {
		t.Fatal("Idle = true with a queued job")
	}
	if _, ok := s.pending(); !ok {
		t.Fatal("pending = !ok, want the job")
	}
	if s.Idle() {
		t.Fatal("Idle = true between pop and run registration; --once would exit before the job starts")
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
		st.Update("s1", &Entry{Status: status, Fingerprint: fp, UpdatedAt: now})
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

func TestParseReply(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                   string
		in                     string
		goal, resp, act, fold  string
	}{
		{"fold line present",
			"ship the omp card\nwrote the digest pass\nnone\nearlier session archived older history",
			"ship the omp card", "wrote the digest pass", "", "earlier session archived older history"},
		{"fold line echoing the prompt contract is dropped",
			"ship the omp card\nwrote the digest pass\nnone\nLine 4: exactly none if no compacted history section; otherwise one line distilling that section.",
			"ship the omp card", "wrote the digest pass", "", ""},
		{"action present",
			"fix the sidebar\nrebuilt card model\napprove the diff",
			"fix the sidebar", "rebuilt card model", "approve the diff", ""},
		{"fold line rides fourth",
			"ship the omp card\nwrote the digest pass\nnone\nfolded history covered card layout",
			"ship the omp card", "wrote the digest pass", "", "folded history covered card layout"},
		{"missing lines map to empty",
			"ship the omp card\nwrote the digest pass",
			"ship the omp card", "wrote the digest pass", "", ""},
		{"only goal",
			"ship the omp card",
			"ship the omp card", "", "", ""},
		{"none lines blank their slot",
			"ship the omp card\nnone\nanswer the question",
			"ship the omp card", "", "answer the question", ""},
		{"none is case-insensitive",
			"ship the omp card\nwrote the digest pass\nNONE",
			"ship the omp card", "wrote the digest pass", "", ""},
		{"empty input",
			"",
			"", "", "", ""},
		{"blank lines skipped",
			"\nship the omp card\n\nwrote the digest pass\n",
			"ship the omp card", "wrote the digest pass", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			goal, resp, act, fold := parseReply(tc.in)
			if goal != tc.goal || resp != tc.resp || act != tc.act || fold != tc.fold {
				t.Fatalf("parseReply(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tc.in, goal, resp, act, fold, tc.goal, tc.resp, tc.act, tc.fold)
			}
		})
	}
}
