package every

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"a", "pi-update", "x_y.z", "abc123", "A-B"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("expected %q valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "with space", "weird/name", "ümlaut", "abc!"} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("expected %q invalid", bad)
		}
	}
}

func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := StatePath(dir, "demo")
	snooze := time.Date(2026, 5, 25, 1, 0, 0, 0, time.UTC)
	in := &State{
		Name:        "demo",
		Command:     "echo hi",
		IntervalSec: 60,
		PID:         os.Getpid(),
		StartedAt:   time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		Status:      StatusHealthy,
		SnoozeUntil: snooze,
	}
	if err := WriteState(p, in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "demo" || out.IntervalSec != 60 || out.Status != StatusHealthy {
		t.Fatalf("round trip mismatch: %+v", out)
	}
	if !out.SnoozeUntil.Equal(snooze) {
		t.Fatalf("SnoozeUntil round trip failed: got %v want %v", out.SnoozeUntil, snooze)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("tmp file leaked")
	}
}

func TestReadStateMissingFields(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.state.json")
	if err := os.WriteFile(p, []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "x" {
		t.Fatalf("expected name x, got %v", st)
	}
}

func TestPIDAlive(t *testing.T) {
	if !PIDAlive(os.Getpid()) {
		t.Fatal("our own pid should be alive")
	}
	if PIDAlive(0) || PIDAlive(-1) {
		t.Fatal("invalid pid reported alive")
	}
	// Best-effort: pid 1 is launchd on macOS, may or may not respond — skip.
}

func TestLifecycleClassification(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	live := &State{Name: "live", PID: os.Getpid(), IntervalSec: 60, LastRunAt: now.Add(-30 * time.Second)}
	if got := Lifecycle(live, time.Time{}, now); got != LifecycleActive {
		t.Errorf("live: got %s", got)
	}
	stopped := &State{Name: "stopped", PID: 0, LastRunAt: now.Add(-3 * 24 * time.Hour)}
	if got := Lifecycle(stopped, time.Time{}, now); got != LifecycleStopped {
		t.Errorf("stopped: got %s", got)
	}
	orph := &State{Name: "orph", PID: 0, LastRunAt: now.Add(-30 * 24 * time.Hour)}
	if got := Lifecycle(orph, time.Time{}, now); got != LifecycleOrphan {
		t.Errorf("orphan: got %s", got)
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b"} {
		if err := WriteState(StatePath(dir, n), &State{Name: n}); err != nil {
			t.Fatal(err)
		}
	}
	// throw in a corrupt one
	_ = os.WriteFile(filepath.Join(dir, "bad.state.json"), []byte("not json"), 0o644)

	states, _, errs, err := LoadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 {
		t.Errorf("want 2 valid states, got %d", len(states))
	}
	if len(errs) != 1 {
		t.Errorf("want 1 corrupt entry, got %d", len(errs))
	}
}
