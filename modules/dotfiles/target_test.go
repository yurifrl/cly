package dotfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func withContext(t *testing.T, user, goos, arch string) {
	t.Helper()
	prev := currentContext
	currentContext = func() (string, string, string) { return user, goos, arch }
	t.Cleanup(func() { currentContext = prev })
}

func TestParseTarget(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantErr bool
		check   func(Target) bool
	}{
		{"user+os", "@target user=yuri-workstation os=linux", false, func(t Target) bool {
			return len(t.Users) == 1 && t.Users[0] == "yuri-workstation" && len(t.OSes) == 1 && t.OSes[0] == "linux"
		}},
		{"csv values", "@target user=yuri,yuri-workstation arch=amd64,arm64", false, func(t Target) bool {
			return len(t.Users) == 2 && len(t.Arches) == 2
		}},
		{"empty", "@target", true, nil},
		{"bad token", "@target user", true, nil},
		{"unknown key", "@target host=box", true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTarget(tc.line, 1)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.line)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.check(got) {
				t.Fatalf("target did not match: %+v", got)
			}
		})
	}
}

func TestTargetGateReason(t *testing.T) {
	withContext(t, "yuri-workstation", "linux", "amd64")

	if r := (Target{}).GateReason(); r != "" {
		t.Fatalf("unset target should permit everything, got %q", r)
	}
	match, _ := parseTarget("@target user=yuri-workstation os=linux", 1)
	if r := match.GateReason(); r != "" {
		t.Fatalf("matching target should permit, got %q", r)
	}
	// root on the same linux box must not be permitted by a yuri-workstation config.
	withContext(t, "root", "linux", "amd64")
	if r := match.GateReason(); r == "" {
		t.Fatal("root must be gated out of a user=yuri-workstation config")
	}
	// the mac default config must not run on linux.
	macDefault, _ := parseTarget("@target os=darwin", 1)
	if r := macDefault.GateReason(); r == "" {
		t.Fatal("os=darwin config must be gated out on linux")
	}
}

func TestParseConfigTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.conf")
	if err := os.WriteFile(path, []byte("@target user=yuri os=darwin\n./a -> ~/a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Errors) != 0 {
		t.Fatalf("unexpected parse errors: %v", cfg.Errors)
	}
	if !cfg.Target.set || len(cfg.Target.Users) != 1 || len(cfg.Target.OSes) != 1 {
		t.Fatalf("target not parsed into config: %+v", cfg.Target)
	}
}

// The base config is always a candidate and the per-user file is an additional
// overlay ordered after it, so the base can never be displaced by the overlay.
func TestConfigCandidatesBaseThenUserOverlay(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "dotfiles.conf")
	userConf := filepath.Join(dir, "dotfiles.yuri-workstation.conf")
	for _, p := range []string{base, userConf} {
		if err := os.WriteFile(p, []byte("@target os=linux\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	prevFlag := configFlag
	configFlag = base
	t.Cleanup(func() { configFlag = prevFlag })

	withContext(t, "yuri-workstation", "linux", "amd64")

	got := configCandidates()
	want := []string{base, userConf}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %v, want %v", got, want)
		}
	}
}

func TestLockPathFor(t *testing.T) {
	cases := map[string]string{
		"/d/dotfiles.conf":                  "/d/dotfiles.lock",
		"/d/dotfiles.yuri-workstation.conf": "/d/dotfiles.yuri-workstation.lock",
	}
	for in, want := range cases {
		if got := lockPathFor(in); got != want {
			t.Fatalf("lockPathFor(%q) = %q, want %q", in, got, want)
		}
	}
}
