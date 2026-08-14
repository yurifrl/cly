package dotfiles

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// A bare @target line used to gate the whole file. That made no sense in
// dotfiles.conf, which always applies, so it is now rejected outright and
// gating is expressed per directive instead.
func TestParseConfig_StandaloneTargetIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dotfiles.conf")
	if err := os.WriteFile(path, []byte("@target user=yuri os=darwin\n./a -> ~/a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Errors) != 1 {
		t.Fatalf("want exactly one error, got %v", cfg.Errors)
	}
	if !strings.Contains(cfg.Errors[0], "@target must be attached to a directive") {
		t.Fatalf("error should explain the inline form, got %q", cfg.Errors[0])
	}
	// The mapping on the following line is unaffected by the bad gate line.
	if len(cfg.Mappings) != 1 {
		for _, m := range cfg.Mappings {
			t.Logf("mapping: %+v", m)
		}
		t.Fatalf("want the ./a mapping to still parse, got %d", len(cfg.Mappings))
	}
}

// Inline @target gates one directive at a time; non-matching lines are dropped
// silently, which is what lets os/arch gating live in a config that always
// applies.
func TestParseConfig_InlineTargetGatesIndividualDirectives(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.conf")
	content := strings.Join([]string{
		"./darwin -> " + filepath.Join(dir, "darwin.out") + " @target os=darwin",
		"./linux -> " + filepath.Join(dir, "linux.out") + " @target os=linux",
		"./always -> " + filepath.Join(dir, "always.out"),
		"!echo darwin @target os=darwin",
		"!echo linux @target os=linux",
		"@cache echo cached @target arch=arm64",
		"@cache echo skipped @target arch=s390x",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	withContext(t, "bob", "darwin", "arm64")

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Errors) != 0 {
		t.Fatalf("unexpected parse errors: %v", cfg.Errors)
	}

	var dests []string
	for _, m := range cfg.Mappings {
		dests = append(dests, filepath.Base(m.Destination))
	}
	wantDests := []string{"darwin.out", "always.out"}
	if !reflect.DeepEqual(dests, wantDests) {
		t.Fatalf("mappings = %v, want %v", dests, wantDests)
	}
	want := []InstallCommand{{Command: "echo darwin", Gate: "@target os=darwin"}}
	if !reflect.DeepEqual(cfg.InstallCommands, want) {
		t.Fatalf("install commands = %+v, want %+v", cfg.InstallCommands, want)
	}
	if len(cfg.CacheEntries) != 1 || cfg.CacheEntries[0].Command != "echo cached" {
		t.Fatalf("cache entries = %+v, want only 'echo cached'", cfg.CacheEntries)
	}
}

// The matching gate is recorded on each entry so output can show why a
// machine-specific line applied; ungated entries keep an empty gate.
func TestParseConfig_MatchedGateIsRecordedOnEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.conf")
	content := strings.Join([]string{
		"./mac -> " + filepath.Join(dir, "mac.out") + " @target os=darwin arch=arm64",
		"./always -> " + filepath.Join(dir, "always.out"),
		"@cache echo c @target os=darwin",
		"@install https://example.com/i.sh @target os=darwin",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	withContext(t, "bob", "darwin", "arm64")

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Errors) != 0 {
		t.Fatalf("unexpected parse errors: %v", cfg.Errors)
	}
	if len(cfg.Mappings) != 2 {
		t.Fatalf("want 2 mappings, got %+v", cfg.Mappings)
	}
	if got, want := cfg.Mappings[0].Gate, "@target os=darwin arch=arm64"; got != want {
		t.Fatalf("gated mapping gate = %q, want %q", got, want)
	}
	if got := cfg.Mappings[1].Gate; got != "" {
		t.Fatalf("ungated mapping must have no gate, got %q", got)
	}
	if got, want := cfg.CacheEntries[0].Gate, "@target os=darwin"; got != want {
		t.Fatalf("cache gate = %q, want %q", got, want)
	}
	if got, want := cfg.Installs[0].Gate, "@target os=darwin"; got != want {
		t.Fatalf("install gate = %q, want %q", got, want)
	}
}

// A gated `!cmd` must be locked under its bare command text; if the gate leaked
// into the identity, the same line would look like a new command per machine
// and re-run forever.
func TestBuildLock_GateIsNotPartOfInstallCommandIdentity(t *testing.T) {
	cfg := &Config{
		InstallCommands: []InstallCommand{{Command: "brew install jq", Gate: "@target os=darwin"}},
	}

	lock := buildLock(cfg, nil)

	want := []string{"brew install jq"}
	if !reflect.DeepEqual(lock.InstallCommands, want) {
		t.Fatalf("lock install commands = %v, want %v", lock.InstallCommands, want)
	}
}

// The gate must not swallow a malformed @target: a typo should surface as an
// error rather than silently dropping the directive it was attached to.
func TestParseConfig_InlineTargetReportsSyntaxErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.conf")
	if err := os.WriteFile(path, []byte("./x -> "+dir+"/x.out @target platform=darwin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Errors) != 1 {
		t.Fatalf("want exactly one error, got %v", cfg.Errors)
	}
	if !strings.Contains(cfg.Errors[0], "unknown @target key") {
		t.Fatalf("error should name the bad key, got %q", cfg.Errors[0])
	}
	if len(cfg.Mappings) != 0 {
		t.Fatalf("a directive with a broken gate must not be applied, got %+v", cfg.Mappings)
	}
}

// A destination containing the literal text "@target" must not be mistaken for
// a gate; only a trailing gate token counts.
func TestParseConfig_InlineTargetOnlySplitsTrailingGate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.conf")
	if err := os.WriteFile(path, []byte("!echo '@target os=darwin is the syntax'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withContext(t, "bob", "darwin", "arm64")

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []InstallCommand{{Command: "echo '@target os=darwin is the syntax'"}}
	if !reflect.DeepEqual(cfg.InstallCommands, want) {
		t.Fatalf("quoted @target must stay part of the command: got %+v, want %+v", cfg.InstallCommands, want)
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
