package ompwrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLaunchArgvWrapsWithHeadroomWhenInstalled(t *testing.T) {
	hr, err := exec.LookPath("headroom")
	if err != nil {
		t.Skip("headroom not installed on this machine")
	}

	got, err := launchArgv(nil)
	if err != nil {
		t.Fatalf("launchArgv(nil): %v", err)
	}
	want := []string{hr, "wrap", "omp"}
	if len(got) != len(want) {
		t.Fatalf("empty rest: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("empty rest: got %v, want %v", got, want)
		}
	}

	got, err = launchArgv([]string{"--model", "opus", "-p", "hi"})
	if err != nil {
		t.Fatalf("launchArgv(rest): %v", err)
	}
	want = []string{hr, "wrap", "omp", "--", "--model", "opus", "-p", "hi"}
	if len(got) != len(want) {
		t.Fatalf("with rest: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("with rest: got %v, want %v", got, want)
		}
	}
}

func TestLaunchArgvFallsBackToPlainOmpWithoutHeadroom(t *testing.T) {
	// PATH containing omp but not headroom.
	dir := t.TempDir()
	ompPath, err := exec.LookPath("omp")
	if err != nil {
		t.Skip("omp not installed on this machine")
	}
	if err := os.Symlink(ompPath, filepath.Join(dir, "omp")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)

	got, err := launchArgv([]string{"--version"})
	if err != nil {
		t.Fatalf("launchArgv: %v", err)
	}
	want := []string{filepath.Join(dir, "omp"), "--version"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLaunchArgvErrorsWhenNeitherInstalled(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-cly-ompwrap-test")

	_, err := launchArgv(nil)
	if err == nil {
		t.Fatal("expected error when neither headroom nor omp on PATH")
	}
}

// TestLaunchArgvRespectsHeadroomConfigToggle covers modules.ompwrap.headroom:
// unset defaults to wrapping; `false` forces direct omp even with headroom on PATH.
func TestLaunchArgvRespectsHeadroomConfigToggle(t *testing.T) {
	ompPath, err := exec.LookPath("omp")
	if err != nil {
		t.Skip("omp not installed on this machine")
	}

	// Isolated HOME with no config file: key unset → default (wrap).
	defaultHome := t.TempDir()
	t.Setenv("HOME", defaultHome)

	got, err := launchArgv([]string{"--version"})
	if err != nil {
		t.Fatalf("launchArgv with key unset: %v", err)
	}
	if filepath.Base(got[0]) != "headroom" || len(got) < 3 || got[1] != "wrap" || got[2] != "omp" {
		t.Fatalf("expected headroom wrap omp by default, got %v", got)
	}

	// Isolated HOME with headroom: false: direct omp.
	disabledHome := t.TempDir()
	configDir := filepath.Join(disabledHome, ".config", "cly")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "modules:\n  ompwrap:\n    headroom: false\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", disabledHome)

	got, err = launchArgv([]string{"--version"})
	if err != nil {
		t.Fatalf("launchArgv disabled: %v", err)
	}
	want := []string{ompPath, "--version"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
