package bundle

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// withTempConfig points the config package at a temp HOME so tests never
// touch the real user config, then writes the given keys.
func withTempConfig(t *testing.T, kv map[string]string) {
	t.Helper()
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "cly")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmp)
	t.Setenv("CLY_APP_CONFIG_DIR", configDir)
	for k, v := range kv {
		if err := pkgconfig.Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
}

// writeFakeManager writes a shell script standing in for the package
// manager. It appends each invocation's args to $CALL_LOG, fails the
// first call when $FAIL_FIRST is set, and always fails when
// $FAIL_ALWAYS is set.
func writeFakeManager(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-manager")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$CALL_LOG"
if [ -n "$FAIL_ALWAYS" ]; then exit 1; fi
if [ -n "$FAIL_FIRST" ] && [ ! -f "$CALL_LOG.done1" ]; then touch "$CALL_LOG.done1"; exit 1; fi
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// readCallLog returns one entry per fake manager invocation.
func readCallLog(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestJsInstallArgs(t *testing.T) {
	tests := []struct {
		name     string
		addCmd   string
		pkg      string
		force    bool
		registry string
		want     []string
	}{
		{
			name:   "no registry no force",
			addCmd: "add",
			pkg:    "@trailhq/trail-code",
			want:   []string{"add", "-g", "@trailhq/trail-code"},
		},
		{
			name:     "with registry",
			addCmd:   "add",
			pkg:      "@trailhq/trail-code",
			registry: "https://registry.npmjs.org/",
			want:     []string{"add", "-g", "@trailhq/trail-code", "--registry=https://registry.npmjs.org/"},
		},
		{
			name:     "registry and force",
			addCmd:   "install",
			pkg:      "pkg@1.0.0",
			force:    true,
			registry: "https://example.com/npm/",
			want:     []string{"install", "-g", "pkg@1.0.0", "--registry=https://example.com/npm/", "--force"},
		},
		{
			name:   "force only",
			addCmd: "add",
			pkg:    "pkg",
			force:  true,
			want:   []string{"add", "-g", "pkg", "--force"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsInstallArgs(tt.addCmd, tt.pkg, tt.force, tt.registry)
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Errorf("jsInstallArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsFallbackRegistry(t *testing.T) {
	t.Run("configured value", func(t *testing.T) {
		withTempConfig(t, map[string]string{
			"modules.bundle.js_fallback_registry": "https://backup.example.com/npm/",
		})
		if got := jsFallbackRegistry(); got != "https://backup.example.com/npm/" {
			t.Errorf("jsFallbackRegistry() = %q, want configured fallback", got)
		}
	})
	t.Run("unset means no fallback", func(t *testing.T) {
		withTempConfig(t, nil)
		if got := jsFallbackRegistry(); got != "" {
			t.Errorf("jsFallbackRegistry() = %q, want empty when unset", got)
		}
	})
}

func TestJsInstallWithFallback_SucceedsNoRetry(t *testing.T) {
	withTempConfig(t, nil)
	log := filepath.Join(t.TempDir(), "calls")
	mgr := writeFakeManager(t)
	t.Setenv("CALL_LOG", log)

	_, note, err := jsInstallWithFallback(mgr, "add", "pkg", false)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty on plain success", note)
	}
	calls := readCallLog(t, log)
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want single attempt", calls)
	}
	if calls[0] != "add -g pkg" {
		t.Errorf("call = %q, want plain install without registry flag", calls[0])
	}
}

func TestJsInstallWithFallback_NoRetryWhenUnset(t *testing.T) {
	withTempConfig(t, nil)
	log := filepath.Join(t.TempDir(), "calls")
	mgr := writeFakeManager(t)
	t.Setenv("CALL_LOG", log)
	t.Setenv("FAIL_FIRST", "1")

	_, note, err := jsInstallWithFallback(mgr, "add", "pkg", false)
	if err == nil {
		t.Fatal("err = nil, want failure")
	}
	if note != "" {
		t.Errorf("note = %q, want empty when no fallback configured", note)
	}
	calls := readCallLog(t, log)
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want single attempt with no fallback configured", calls)
	}
	if calls[0] != "add -g pkg" {
		t.Errorf("call = %q, want plain install without registry flag", calls[0])
	}
}

func TestJsInstallWithFallback_RetriesWithConfigured(t *testing.T) {
	configured := "https://backup.example.com/npm/"
	withTempConfig(t, map[string]string{
		"modules.bundle.js_fallback_registry": configured,
	})
	log := filepath.Join(t.TempDir(), "calls")
	mgr := writeFakeManager(t)
	t.Setenv("CALL_LOG", log)
	t.Setenv("FAIL_FIRST", "1")

	_, note, err := jsInstallWithFallback(mgr, "add", "pkg", false)
	if err != nil {
		t.Fatalf("err = %v, want fallback retry success", err)
	}
	if !strings.Contains(note, configured) || !strings.Contains(note, "failed on the default registry") {
		t.Errorf("note = %q, want user-visible fallback notice mentioning %s", note, configured)
	}
	calls := readCallLog(t, log)
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want 2 attempts", calls)
	}
	if calls[0] != "add -g pkg" {
		t.Errorf("first attempt = %q, want plain install without registry flag", calls[0])
	}
	wantRetry := "add -g pkg --registry=" + configured
	if calls[1] != wantRetry {
		t.Errorf("retry = %q, want %q", calls[1], wantRetry)
	}
}

func TestJsInstallWithFallback_BothFail(t *testing.T) {
	configured := "https://backup.example.com/npm/"
	withTempConfig(t, map[string]string{
		"modules.bundle.js_fallback_registry": configured,
	})
	log := filepath.Join(t.TempDir(), "calls")
	mgr := writeFakeManager(t)
	t.Setenv("CALL_LOG", log)
	t.Setenv("FAIL_ALWAYS", "1")

	_, note, err := jsInstallWithFallback(mgr, "add", "pkg", false)
	if err == nil {
		t.Fatal("err = nil, want failure")
	}
	if note != "" {
		t.Errorf("note = %q, want empty when both attempts fail", note)
	}
	if !strings.Contains(err.Error(), "also failed") {
		t.Errorf("err = %v, want both attempts reported", err)
	}
	calls := readCallLog(t, log)
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want 2 attempts", calls)
	}
	if calls[0] != "add -g pkg" {
		t.Errorf("first attempt = %q, want plain install without registry flag", calls[0])
	}
	wantRetry := "add -g pkg --registry=" + configured
	if calls[1] != wantRetry {
		t.Errorf("retry = %q, want %q", calls[1], wantRetry)
	}
}

func TestParseBunTree(t *testing.T) {
	output := `/Users/y/.bun/install/global/node_modules
├── typescript@5.9.3
├── @types/node@24.7.2
└── undici@7.16.0 (v7.16.0 available)
`
	got := parseBunTree(output)
	want := map[string]string{
		"typescript":  "5.9.3",
		"@types/node": "24.7.2",
		"undici":      "7.16.0",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries (%v), want %d", len(got), got, len(want))
	}
	for base, version := range want {
		if got[base] != version {
			t.Errorf("got[%q] = %q, want %q", base, got[base], version)
		}
	}
}

func TestRegistryFromNpmrcLike(t *testing.T) {
	dir := t.TempDir()
	npmrc := filepath.Join(dir, ".npmrc")
	bunfig := filepath.Join(dir, ".bunfig.toml")
	if err := os.WriteFile(bunfig, []byte("install.registry = \"https://bun.example.com\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// bunfig.toml form: install.registry = value.
	if got := registryFromNpmrcLike(bunfig); got != "https://bun.example.com" {
		t.Errorf("bunfig = %q", got)
	}
	// .npmrc form: registry=value.
	if err := os.WriteFile(npmrc, []byte("registry=https://npm.example.com/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := registryFromNpmrcLike(npmrc); got != "https://npm.example.com/" {
		t.Errorf("npmrc = %q", got)
	}
	// Missing file -> empty.
	if got := registryFromNpmrcLike(filepath.Join(dir, "missing")); got != "" {
		t.Errorf("missing file = %q, want empty", got)
	}
}

func TestJsDefaultRegistry(t *testing.T) {
	t.Setenv("npm_config_registry", "")
	// Unset npm env, no config files -> npm default.
	withTempConfig(t, nil)
	if got := jsDefaultRegistry(); got != "https://registry.npmjs.org/" {
		t.Errorf("default = %q", got)
	}
	// bunfig wins when present.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	writeFile(t, filepath.Join(tmp, ".bunfig.toml"),
		"install.registry = \"https://mirror.example.com\"\n")
	if got := jsDefaultRegistry(); got != "https://mirror.example.com" {
		t.Errorf("bunfig = %q", got)
	}
	// Env var wins over bunfig.
	t.Setenv("npm_config_registry", "https://env.example.com")
	if got := jsDefaultRegistry(); got != "https://env.example.com" {
		t.Errorf("env = %q", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// jsRegistryStub serves dist-tag manifests for /pkg/tag URLs (scoped names
// arrive escaped as /@scope%2Fpkg/tag).
func jsRegistryStub(t *testing.T, tags map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is fully decoded (%2F becomes "/"); read RawPath so
		// scoped names arrive as "@scope%2Fpkg".
		path := r.URL.Path
		if r.URL.RawPath != "" {
			path = r.URL.RawPath
		}
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		pkg, err := url.PathUnescape(parts[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		version, ok := tags[pkg+"/"+parts[1]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"version":%q}`, version)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPlanJsSync(t *testing.T) {
	t.Setenv("npm_config_registry", "")
	srv := jsRegistryStub(t, map[string]string{
		"lodash/latest":      "4.17.21",
		"lodash/next":        "5.0.0",
		"@types/node/latest": "24.7.2",
	})
	tests := []struct {
		name      string
		desired   []string
		installed map[string]string
		want      map[string]string // base -> action.verb
	}{
		{
			name:    "absent installs",
			desired: []string{"lodash@4.17.20"},
			want:    map[string]string{"lodash": "install"},
		},
		{
			name:      "equal pin skips",
			desired:   []string{"lodash@4.17.21"},
			installed: map[string]string{"lodash": "4.17.21"},
			want:      map[string]string{},
		},
		{
			name:      "newer installed than pin downgrades",
			desired:   []string{"lodash@4.17.20"},
			installed: map[string]string{"lodash": "4.17.21"},
			want:      map[string]string{"lodash": "downgrade"},
		},
		{
			name:      "older installed than pin upgrades",
			desired:   []string{"lodash@4.17.21"},
			installed: map[string]string{"lodash": "4.0.0"},
			want:      map[string]string{"lodash": "upgrade"},
		},
		{
			name:      "bare up-to-date skips via registry",
			desired:   []string{"@types/node"},
			installed: map[string]string{"@types/node": "24.7.2"},
			want:      map[string]string{},
		},
		{
			name:      "bare outdated upgrades via registry",
			desired:   []string{"lodash"},
			installed: map[string]string{"lodash": "4.0.0"},
			want:      map[string]string{"lodash": "upgrade"},
		},
		{
			name:      "floating tag upgrades via registry",
			desired:   []string{"lodash@next"},
			installed: map[string]string{"lodash": "4.17.21"},
			want:      map[string]string{"lodash": "upgrade"},
		},
		{
			name:      "bare absent installs without registry call",
			desired:   []string{"typescript"},
			installed: map[string]string{},
			want:      map[string]string{"typescript": "install"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, _, err := planJsSync(tt.desired, tt.installed, false, srv.URL)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			got := make(map[string]string, len(actions))
			for _, a := range actions {
				got[a.base] = a.verb
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for base, verb := range tt.want {
				if got[base] != verb {
					t.Errorf("got[%q] = %q, want %q", base, got[base], verb)
				}
			}
		})
	}
}

func TestPlanJsSync_OfflineLeavesBareUntouched(t *testing.T) {
	t.Setenv("npm_config_registry", "")
	installed := map[string]string{"lodash": "4.0.0"}
	actions, _, err := planJsSync([]string{"lodash"}, installed, false, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("err = nil, want registry-failure report")
	}
	if !strings.Contains(err.Error(), "could not check latest versions of lodash") {
		t.Errorf("err = %v, want failed packages named", err)
	}
	if len(actions) != 0 {
		t.Errorf("actions = %v, want none (installed version stays untouched)", actions)
	}
}

func TestPlanJsSync_ForceReinstalls(t *testing.T) {
	t.Setenv("npm_config_registry", "")
	actions, _, err := planJsSync([]string{"lodash@4.17.21"}, map[string]string{"lodash": "4.17.21"}, true, "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(actions) != 1 || actions[0].verb != "reinstall" {
		t.Errorf("actions = %v, want reinstall", actions)
	}
}

func TestJsSyncSummary(t *testing.T) {
	actions := []syncAction{
		{base: "a", verb: "install"},
		{base: "b", verb: "upgrade"},
		{base: "c", verb: "downgrade"},
	}
	if got := syncSummary(actions, 3); got != "1 installed, 1 upgraded, 1 downgraded" {
		t.Errorf("summary = %q", got)
	}
	if got := syncSummary(nil, 5); got != "Nothing changed - all 5 packages up to date" {
		t.Errorf("idle summary = %q", got)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.10.0", "1.9.9", 1},
		{"4.17.21", "4.17.21", 0},
		{"2.0.0", "2.0.0-beta.1", 1},
		{"5.9.3", "6.0.0", -1},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
