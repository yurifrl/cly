package ompsummary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRoundtrip(t *testing.T) {
	cwd := t.TempDir()
	st := LoadState(cwd) // missing file → empty
	if len(st.Summaries) != 0 {
		t.Fatalf("fresh state should be empty, got %d", len(st.Summaries))
	}

	st.Update("s1", &Summary{Text: "building auth", Status: "ok", Fingerprint: "fp1", Provider: "openai", Model: "gpt-5"})
	if err := SaveState(cwd, st); err != nil {
		t.Fatal(err)
	}

	got := LoadState(cwd).Summaries["s1"]
	if got == nil || got.Text != "building auth" || got.Status != "ok" ||
		got.Fingerprint != "fp1" || got.Provider != "openai" || got.Model != "gpt-5" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if got.UpdatedAt == 0 {
		t.Fatal("Update should stamp UpdatedAt")
	}
}

func TestLoadCorruptStateSelfHeals(t *testing.T) {
	cwd := t.TempDir()
	p := StatePath(cwd)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := LoadState(cwd)
	if len(st.Summaries) != 0 {
		t.Fatalf("corrupt file should self-heal to empty, got %d entries", len(st.Summaries))
	}
}

func TestPruneDropsOldEntries(t *testing.T) {
	st := &State{Summaries: map[string]*Summary{}}
	now := nowUnix()
	st.Update("old", &Summary{Status: "ok"})
	st.Summaries["old"].UpdatedAt = now - 7200
	st.Update("new", &Summary{Status: "ok"})

	st.Prune(now, 3600)
	if _, ok := st.Summaries["old"]; ok {
		t.Fatal("old entry should be pruned")
	}
	if _, ok := st.Summaries["new"]; !ok {
		t.Fatal("fresh entry should survive")
	}
}

func TestStatePath(t *testing.T) {
	got := StatePath("/tmp/p")
	if filepath.ToSlash(got) != "/tmp/p/.omp/summary.json" {
		t.Fatalf("StatePath = %q", got)
	}
}
