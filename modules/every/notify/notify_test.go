package notify

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	pkgnotify "github.com/yurifrl/cly/pkg/notify"
)

type fakeNotifier struct {
	mu        sync.Mutex
	available bool
	got       []pkgnotify.Notification
}

func (f *fakeNotifier) Send(_ context.Context, n pkgnotify.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, n)
	return nil
}
func (f *fakeNotifier) Available() bool { return f.available }

func TestSend_RoutesGroupAndSound_Failing(t *testing.T) {
	f := &fakeNotifier{available: true}
	SetShared(f)

	if err := Send(context.Background(), Notification{
		TaskName: "smoke",
		Level:    LevelFailing,
		Title:    "T",
		Body:     "B",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	if len(f.got) != 1 {
		t.Fatalf("want 1 send, got %d", len(f.got))
	}
	n := f.got[0]
	if n.Group != "cly.every.smoke" {
		t.Errorf("group: %q", n.Group)
	}
	if n.Sound != "Basso" {
		t.Errorf("sound: %q", n.Sound)
	}
}

func TestSend_NoCrashWhenUnavailable(t *testing.T) {
	f := &fakeNotifier{available: false}
	SetShared(f)
	if err := Send(context.Background(), Notification{TaskName: "x", Level: LevelFailing}); err != nil {
		t.Fatalf("expected nil error when unavailable, got %v", err)
	}
	if len(f.got) != 0 {
		t.Fatalf("should not send when unavailable")
	}
}

func TestSoundOverride(t *testing.T) {
	cfgDir := t.TempDir()
	old := ConfigDir
	defer func() { ConfigDir = old }()
	ConfigDir = func() (string, error) { return cfgDir, nil }

	if err := os.WriteFile(filepath.Join(cfgDir, "sounds.toml"), []byte("failing=Funk\nrecovered=Hero\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveSound(LevelFailing); got != "Funk" {
		t.Fatalf("failing sound: %s", got)
	}
	if got := resolveSound(LevelRecovered); got != "Hero" {
		t.Fatalf("recovered sound: %s", got)
	}
	if got := resolveSound(LevelGaveUp); got != "Sosumi" {
		t.Fatalf("gaveup default: %s", got)
	}
}

func TestGroupFor(t *testing.T) {
	if g := GroupFor("foo"); g != "cly.every.foo" {
		t.Fatalf("group: %s", g)
	}
}

func TestLevelString(t *testing.T) {
	if LevelFailing.String() != "failing" || LevelRecovered.String() != "recovered" || LevelGaveUp.String() != "gaveup" {
		t.Fatal("level strings drifted")
	}
}
