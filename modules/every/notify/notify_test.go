package notify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestSendMissingTerminalNotifier(t *testing.T) {
	old := LookPath
	defer func() { LookPath = old }()
	LookPath = func(string) (string, error) { return "", errors.New("not found") }
	if err := Send(context.Background(), Notification{TaskName: "x", Title: "t", Body: "b"}); err != nil {
		t.Fatalf("expected nil error when terminal-notifier missing, got %v", err)
	}
}

func TestSendInvokesRunner(t *testing.T) {
	old := LookPath
	defer func() { LookPath = old }()
	LookPath = func(string) (string, error) { return "/usr/bin/true", nil }

	var mu sync.Mutex
	var got []string
	oldRunner := Runner
	defer func() { Runner = oldRunner }()
	Runner = func(ctx context.Context, name string, args ...string) error {
		mu.Lock()
		defer mu.Unlock()
		got = append([]string{name}, args...)
		return nil
	}

	if err := Send(context.Background(), Notification{TaskName: "smoke", Level: LevelFailing, Title: "T", Body: "B"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(got) == 0 || got[0] != "terminal-notifier" {
		t.Fatalf("unexpected runner invocation: %v", got)
	}
	if !contains(got, "-group") || !contains(got, "cly.every.smoke") {
		t.Fatalf("missing -group: %v", got)
	}
	if !contains(got, "-sound") || !contains(got, "Basso") {
		t.Fatalf("missing default Basso sound: %v", got)
	}
}

func TestIconExtractionIdempotent(t *testing.T) {
	old := ConfigDir
	defer func() { ConfigDir = old }()
	ConfigDir = func() (string, error) { return t.TempDir(), nil }

	p1, err := iconPath(LevelFailing)
	if err != nil {
		t.Fatal(err)
	}
	stat1, _ := os.Stat(p1)
	p2, err := iconPath(LevelFailing)
	if err != nil {
		t.Fatal(err)
	}
	stat2, _ := os.Stat(p2)
	if p1 != p2 {
		t.Fatalf("expected same path: %s vs %s", p1, p2)
	}
	if stat1.ModTime() != stat2.ModTime() {
		t.Fatalf("file rewritten between calls")
	}
}

func TestIconOverride(t *testing.T) {
	cfgDir := t.TempDir()
	old := ConfigDir
	defer func() { ConfigDir = old }()
	ConfigDir = func() (string, error) { return cfgDir, nil }

	override := filepath.Join(cfgDir, "icons", "failing.png")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := iconPath(LevelFailing)
	if err != nil {
		t.Fatal(err)
	}
	if got != override {
		t.Fatalf("expected override %s, got %s", override, got)
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

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle || strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
