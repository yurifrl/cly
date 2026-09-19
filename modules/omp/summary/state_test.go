package ompsummary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateRoundtrip(t *testing.T) {
	cwd := t.TempDir()
	st := LoadState(cwd) // missing file → empty
	if len(st.Sessions) != 0 {
		t.Fatalf("fresh state should be empty, got %d", len(st.Sessions))
	}

	st.Update("s1", &Entry{
		Goal:        "building auth",
		AIStatus:    "editing login.go",
		Status:      StatusOK,
		Fingerprint: "fp1",
		Provider:    "openai",
		Model:       "gpt-5",
	})
	if err := SaveState(cwd, st); err != nil {
		t.Fatal(err)
	}

	got := LoadState(cwd).Get("s1")
	if got == nil || got.Goal != "building auth" || got.AIStatus != "editing login.go" ||
		got.Status != StatusOK || got.Fingerprint != "fp1" ||
		got.Provider != "openai" || got.Model != "gpt-5" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if got.UpdatedAt == 0 {
		t.Fatal("Update should stamp UpdatedAt")
	}

	// On-disk shape is v2: version + sessions key.
	b, err := os.ReadFile(StatePath(cwd))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"version": 2`) || !strings.Contains(string(b), `"sessions"`) {
		t.Fatalf("saved file not v2 shape:\n%s", b)
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
	if len(st.Sessions) != 0 {
		t.Fatalf("corrupt file should self-heal to empty, got %d entries", len(st.Sessions))
	}
}

func TestV1MigrationPreservesGoalAndModel(t *testing.T) {
	cwd := t.TempDir()
	p := StatePath(cwd)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	v1 := `{"summaries":{"s1":{"text":"building auth","status":"ok","model":"gpt-5","provider":"openai","fingerprint":"fp1","updated_at_unix":100}}}`
	if err := os.WriteFile(p, []byte(v1), 0o644); err != nil {
		t.Fatal(err)
	}
	got := LoadState(cwd).Get("s1")
	if got == nil {
		t.Fatal("v1 entry not migrated")
	}
	if got.Goal != "building auth" || got.Model != "gpt-5" || got.Provider != "openai" || got.Fingerprint != "fp1" {
		t.Fatalf("migration lost fields: %+v", got)
	}
	if got.Exit != ExitUnknown {
		t.Fatalf("migrated Exit = %d, want ExitUnknown", got.Exit)
	}

	// Saving migrates the file to v2 permanently.
	if err := SaveState(cwd, LoadState(cwd)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"version": 2`) {
		t.Fatalf("save after migration should be v2:\n%s", b)
	}
}

func TestPruneDropsOldEntries(t *testing.T) {
	st := &State{Sessions: map[string]*Entry{}}
	now := nowUnix()
	st.Update("old", &Entry{Status: StatusOK})
	st.Sessions["old"].UpdatedAt = now - 7200
	st.Update("new", &Entry{Status: StatusOK})

	st.Prune(now, 3600)
	if _, ok := st.Sessions["old"]; ok {
		t.Fatal("old entry should be pruned")
	}
	if _, ok := st.Sessions["new"]; !ok {
		t.Fatal("fresh entry should survive")
	}
}

func TestNewerThan(t *testing.T) {
	st := &State{Sessions: map[string]*Entry{}}
	if st.NewerThan(nowUnix()) {
		t.Fatal("empty state is never newer")
	}
	mark := nowUnix() - 10
	st.Update("s1", &Entry{Status: StatusOK})
	if st.Sessions["s1"].UpdatedAt <= mark {
		t.Fatal("test setup: entry should postdate mark")
	}
	if !st.NewerThan(mark) {
		t.Fatal("entry newer than mark → NewerThan true")
	}
	if st.NewerThan(st.Sessions["s1"].UpdatedAt) {
		t.Fatal("strictly-older mark → NewerThan false")
	}
}

func TestStatePath(t *testing.T) {
	got := StatePath("/tmp/p")
	if filepath.ToSlash(got) != "/tmp/p/.omp/summary.json" {
		t.Fatalf("StatePath = %q", got)
	}
}
