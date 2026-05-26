package every

import (
	"os"
	"testing"
	"time"
)

func TestPruneOrphan(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)

	// orphan: dead pid + last_run > 7d
	WriteState(StatePath(dir, "orph"), &State{Name: "orph", PID: 0, LastRunAt: now.Add(-30 * 24 * time.Hour)})
	// stopped: dead pid + recent
	WriteState(StatePath(dir, "stopped"), &State{Name: "stopped", PID: 0, LastRunAt: now.Add(-2 * 24 * time.Hour)})
	// active: live pid (use our own pid)
	WriteState(StatePath(dir, "active"), &State{Name: "active", PID: os.Getpid(), LastRunAt: now, IntervalSec: 60})
	// touch a log file we'll expect removed for orphan
	os.WriteFile(LogPath(dir, "orph"), []byte("{}\n"), 0o644)

	// dry run
	res, err := Prune(dir, now, PruneOptions{Apply: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Categories.Orphan) != 1 || res.Categories.Orphan[0] != "orph" {
		t.Fatalf("orphan miscount: %+v", res.Categories)
	}
	if len(res.Removed) != 0 {
		t.Fatalf("dry-run should not remove: %v", res.Removed)
	}

	// apply
	res, err = Prune(dir, now, PruneOptions{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 1 {
		t.Fatalf("expected 1 removed, got %v", res.Removed)
	}
	if _, err := os.Stat(StatePath(dir, "orph")); !os.IsNotExist(err) {
		t.Fatal("orphan state file not removed")
	}
	if _, err := os.Stat(StatePath(dir, "stopped")); err != nil {
		t.Fatal("stopped should be kept")
	}
	if _, err := os.Stat(StatePath(dir, "active")); err != nil {
		t.Fatal("active should be kept")
	}
}

func TestPruneIncludeStopped(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	WriteState(StatePath(dir, "stopped"), &State{Name: "stopped", PID: 0, LastRunAt: now.Add(-2 * 24 * time.Hour)})
	WriteState(StatePath(dir, "active"), &State{Name: "active", PID: os.Getpid(), LastRunAt: now, IntervalSec: 60})

	res, err := Prune(dir, now, PruneOptions{Apply: true, IncludeStopped: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 1 || res.Removed[0] != "stopped" {
		t.Fatalf("expected stopped removed, got %v", res.Removed)
	}
}

func TestSweepOrphans(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	WriteState(StatePath(dir, "orph"), &State{Name: "orph", PID: 0, LastRunAt: now.Add(-30 * 24 * time.Hour)})
	WriteState(StatePath(dir, "stopped"), &State{Name: "stopped", PID: 0, LastRunAt: now.Add(-2 * 24 * time.Hour)})

	removed, err := SweepOrphans(dir, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "orph" {
		t.Fatalf("sweep: %v", removed)
	}
	if _, err := os.Stat(StatePath(dir, "stopped")); err != nil {
		t.Fatal("stopped removed by sweep — should be orphan-only")
	}
}
