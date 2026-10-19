package bundle

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// writeFakeGo writes a shell script standing in for `go`. It appends each
// invocation's spec plus its GOPROXY/GOSUMDB values to $CALL_LOG, fails the
// first call when $FAIL_FIRST is set, and always fails when $FAIL_ALWAYS
// is set.
func writeFakeGo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
printf 'install %s GOPROXY=%s GOSUMDB=%s\n' "$2" "$GOPROXY" "$GOSUMDB" >> "$CALL_LOG"
if [ -n "$FAIL_ALWAYS" ]; then echo "go: proxy rejected the package" >&2; exit 1; fi
if [ -n "$FAIL_FIRST" ] && [ ! -f "$CALL_LOG.done1" ]; then touch "$CALL_LOG.done1"; echo "go: proxy rejected the package" >&2; exit 1; fi
exit 0
`
	path := filepath.Join(dir, "go")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestGoFallbackEnv(t *testing.T) {
	t.Run("returns configured KEY=VALUE pairs", func(t *testing.T) {
		withTempConfig(t, map[string]string{
			"modules.bundle.go_fallback_env": "GOPROXY=direct GOSUMDB=sum.golang.org",
		})
		env := goFallbackEnv()
		if len(env) != 2 || env[0] != "GOPROXY=direct" || env[1] != "GOSUMDB=sum.golang.org" {
			t.Errorf("goFallbackEnv() = %v, want [GOPROXY=direct GOSUMDB=sum.golang.org]", env)
		}
	})

	t.Run("drops malformed entries", func(t *testing.T) {
		withTempConfig(t, map[string]string{
			"modules.bundle.go_fallback_env": "GOPROXY=direct OOPS",
		})
		env := goFallbackEnv()
		if len(env) != 1 || env[0] != "GOPROXY=direct" {
			t.Errorf("goFallbackEnv() = %v, want [GOPROXY=direct]", env)
		}
	})

	t.Run("empty when unset", func(t *testing.T) {
		withTempConfig(t, nil)
		if env := goFallbackEnv(); len(env) != 0 {
			t.Errorf("goFallbackEnv() = %v, want empty", env)
		}
	})
}

func TestGoInstallWithFallback_PlainSuccess(t *testing.T) {
	withTempConfig(t, nil)
	log := filepath.Join(t.TempDir(), "calls")
	writeFakeGo(t)
	t.Setenv("CALL_LOG", log)

	note, err := goInstallWithFallback("mvdan.cc/gofumpt@latest")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty on plain success", note)
	}
	calls := readCallLog(t, log)
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want 1 attempt", calls)
	}
	if !strings.Contains(calls[0], "install mvdan.cc/gofumpt@latest") {
		t.Errorf("call = %q, want install spec", calls[0])
	}
}

func TestGoInstallWithFallback_NoRetryWhenUnset(t *testing.T) {
	withTempConfig(t, nil)
	log := filepath.Join(t.TempDir(), "calls")
	writeFakeGo(t)
	t.Setenv("CALL_LOG", log)
	t.Setenv("FAIL_FIRST", "1")

	_, err := goInstallWithFallback("mvdan.cc/gofumpt@latest")
	if err == nil {
		t.Fatal("err = nil, want failure")
	}
	if !strings.Contains(err.Error(), "proxy rejected the package") {
		t.Errorf("err = %v, want original failure", err)
	}
	if calls := readCallLog(t, log); len(calls) != 1 {
		t.Fatalf("calls = %v, want exactly 1 attempt with no fallback configured", calls)
	}
}

func TestGoInstallWithFallback_RetriesWithEnvOverrides(t *testing.T) {
	configured := "GOPROXY=direct GOSUMDB=sum.golang.org"
	withTempConfig(t, map[string]string{
		"modules.bundle.go_fallback_env": configured,
	})
	log := filepath.Join(t.TempDir(), "calls")
	writeFakeGo(t)
	t.Setenv("CALL_LOG", log)
	t.Setenv("FAIL_FIRST", "1")
	// Ambient machine-wide values that break the first attempt.
	t.Setenv("GOPROXY", "http://127.0.0.1:9")
	t.Setenv("GOSUMDB", "off")

	note, err := goInstallWithFallback("mvdan.cc/gofumpt@latest")
	if err != nil {
		t.Fatalf("err = %v, want fallback retry success", err)
	}
	if !strings.Contains(note, "mvdan.cc/gofumpt@latest") || !strings.Contains(note, "failed on the default GOPROXY") {
		t.Errorf("note = %q, want user-visible fallback notice", note)
	}
	if !strings.Contains(note, "GOPROXY=direct") || !strings.Contains(note, "GOSUMDB=sum.golang.org") {
		t.Errorf("note = %q, want the exact fallback env used", note)
	}
	calls := readCallLog(t, log)
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want 2 attempts", calls)
	}
	if !strings.Contains(calls[0], "GOPROXY=http://127.0.0.1:9") || !strings.Contains(calls[0], "GOSUMDB=off") {
		t.Errorf("first attempt = %q, want ambient env unchanged", calls[0])
	}
	if !strings.Contains(calls[1], "GOPROXY=direct") || !strings.Contains(calls[1], "GOSUMDB=sum.golang.org") {
		t.Errorf("retry = %q, want fallback overrides applied", calls[1])
	}
}

func TestGoInstallWithFallback_BothFail(t *testing.T) {
	withTempConfig(t, map[string]string{
		"modules.bundle.go_fallback_env": "GOPROXY=direct",
	})
	log := filepath.Join(t.TempDir(), "calls")
	writeFakeGo(t)
	t.Setenv("CALL_LOG", log)
	t.Setenv("FAIL_ALWAYS", "1")

	note, err := goInstallWithFallback("mvdan.cc/gofumpt@latest")
	if err == nil {
		t.Fatal("err = nil, want failure")
	}
	if note != "" {
		t.Errorf("note = %q, want empty when both attempts fail", note)
	}
	if !strings.Contains(err.Error(), "also failed") || !strings.Contains(err.Error(), "GOPROXY=direct") {
		t.Errorf("err = %v, want both attempts reported with fallback env", err)
	}
	if calls := readCallLog(t, log); len(calls) != 2 {
		t.Fatalf("calls = %v, want 2 attempts", calls)
	}
}

func TestParseGoPkg(t *testing.T) {
	tests := []struct {
		spec     string
		expected string
		version  string
	}{
		{"mvdan.cc/gofumpt", "mvdan.cc/gofumpt", ""},
		{"mvdan.cc/gofumpt@latest", "mvdan.cc/gofumpt", "latest"},
		{"golang.org/x/tools/cmd/goimports@v0.24.0", "golang.org/x/tools/cmd/goimports", "v0.24.0"},
		{"github.com/rakyll/hey@v0.1.12", "github.com/rakyll/hey", "v0.1.12"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			spec, version := parseGoPkg(tt.spec)
			assert.Equal(t, tt.expected, spec)
			assert.Equal(t, tt.version, version)
		})
	}
}

func TestMatchGoTool(t *testing.T) {
	tool := goTool{
		binary:  "/Users/x/go/bin/goimports",
		module:  "golang.org/x/tools",
		version: "v0.24.0",
	}
	moduleTool := goTool{
		binary:  "/Users/x/go/bin/gofumpt",
		module:  "mvdan.cc/gofumpt",
		version: "v0.7.0",
	}

	tests := []struct {
		spec     string
		tool     goTool
		expected bool
	}{
		// Module root tools: spec is the module path.
		{"mvdan.cc/gofumpt", moduleTool, true},
		{"mvdan.cc/gofumpt@v0.7.0", moduleTool, true},
		// Main packages inside a multi-package module: binary name match.
		{"golang.org/x/tools/cmd/goimports", tool, true},
		{"golang.org/x/tools/cmd/goimports@v0.24.0", tool, true},
		// Wrong command inside the module.
		{"golang.org/x/tools/cmd/stringer", tool, false},
		// Wrong module.
		{"mvdan.cc/sh", tool, false},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			spec, _ := parseGoPkg(tt.spec)
			assert.Equal(t, tt.expected, matchGoTool(spec, tt.tool))
		})
	}
}

func TestFindGoTool(t *testing.T) {
	tools := []goTool{
		{binary: "/Users/x/go/bin/goimports", module: "golang.org/x/tools", version: "v0.24.0"},
		{binary: "/Users/x/go/bin/gofumpt", module: "mvdan.cc/gofumpt", version: "v0.7.0"},
	}

	t.Run("finds module-root tool", func(t *testing.T) {
		tool, found := findGoTool("mvdan.cc/gofumpt", tools)
		assert.True(t, found)
		assert.Equal(t, "v0.7.0", tool.version)
	})

	t.Run("finds main-package tool", func(t *testing.T) {
		tool, found := findGoTool("golang.org/x/tools/cmd/goimports@v0.24.0", tools)
		assert.True(t, found)
		assert.Equal(t, "v0.24.0", tool.version)
	})

	t.Run("misses unknown spec", func(t *testing.T) {
		_, found := findGoTool("github.com/nobody/nowhere", tools)
		assert.False(t, found)
	})
}

// proxyServer serves {"Version": version} for every @latest request, counting
// hits. Handlers can override per-path behavior.
func proxyServer(t *testing.T, version string, handler func(w http.ResponseWriter, r *http.Request, hits int)) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if handler != nil {
			handler(w, r, hits)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"Version": %q}`, version)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GOPROXY", srv.URL)
	return srv, &hits
}

func TestPlanGoSync(t *testing.T) {
	tool := func(module, version string) goTool {
		return goTool{binary: "/go/bin/" + filepath.Base(module), module: module, version: version}
	}

	t.Run("installs missing pinned spec", func(t *testing.T) {
		actions, examined, err := planGoSync([]string{"github.com/rakyll/hey@v0.1.12"}, nil, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Equal(t, []syncAction{{pkg: "github.com/rakyll/hey@v0.1.12", base: "hey", verb: "install", to: "v0.1.12"}}, actions)
	})

	t.Run("pinned equal is skipped", func(t *testing.T) {
		actions, examined, err := planGoSync([]string{"mvdan.cc/gofumpt@v0.7.0"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Empty(t, actions)
	})

	t.Run("pinned pin without v prefix matches installed", func(t *testing.T) {
		actions, examined, err := planGoSync([]string{"mvdan.cc/gofumpt@0.7.0"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Empty(t, actions)
	})

	t.Run("pinned move up upgrades", func(t *testing.T) {
		actions, examined, err := planGoSync([]string{"mvdan.cc/gofumpt@v0.8.0"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Equal(t, []syncAction{{pkg: "mvdan.cc/gofumpt@v0.8.0", base: "gofumpt", verb: "upgrade", from: "v0.7.0", to: "v0.8.0"}}, actions)
	})

	t.Run("pinned move down downgrades", func(t *testing.T) {
		actions, _, err := planGoSync([]string{"mvdan.cc/gofumpt@v0.6.0"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.NoError(t, err)
		assert.Equal(t, []syncAction{{pkg: "mvdan.cc/gofumpt@v0.6.0", base: "gofumpt", verb: "downgrade", from: "v0.7.0", to: "v0.6.0"}}, actions)
	})

	t.Run("devel build moves to pin as upgrade", func(t *testing.T) {
		actions, _, err := planGoSync([]string{"mvdan.cc/gofumpt@v0.8.0"}, []goTool{tool("mvdan.cc/gofumpt", "(devel)")}, false)
		assert.NoError(t, err)
		assert.Equal(t, []syncAction{{pkg: "mvdan.cc/gofumpt@v0.8.0", base: "gofumpt", verb: "upgrade", from: "(devel)", to: "v0.8.0"}}, actions)
	})

	t.Run("force reinstalls equal pinned version", func(t *testing.T) {
		actions, _, err := planGoSync([]string{"mvdan.cc/gofumpt@v0.7.0"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, true)
		assert.NoError(t, err)
		assert.Equal(t, []syncAction{{pkg: "mvdan.cc/gofumpt@v0.7.0", base: "gofumpt", verb: "reinstall"}}, actions)
	})

	t.Run("force keeps move labels", func(t *testing.T) {
		actions, _, err := planGoSync([]string{"mvdan.cc/gofumpt@v0.6.0"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, true)
		assert.NoError(t, err)
		assert.Equal(t, "downgrade", actions[0].verb)
	})

	t.Run("duplicate specs dedupe", func(t *testing.T) {
		_, examined, err := planGoSync([]string{"github.com/rakyll/hey@v0.1.12", "github.com/rakyll/hey@v0.1.12"}, nil, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
	})

	t.Run("bare missing tool never hits proxy", func(t *testing.T) {
		proxyServer(t, "v9.9.9", nil)
		actions, examined, err := planGoSync([]string{"github.com/rakyll/hey"}, nil, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Equal(t, []syncAction{{pkg: "github.com/rakyll/hey", base: "hey", verb: "install"}}, actions)
	})

	t.Run("bare installed upgrades to proxy latest", func(t *testing.T) {
		proxyServer(t, "v0.9.0", nil)
		actions, examined, err := planGoSync([]string{"mvdan.cc/gofumpt"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Equal(t, []syncAction{{pkg: "mvdan.cc/gofumpt", base: "gofumpt", verb: "upgrade", from: "v0.7.0", to: "v0.9.0"}}, actions)
	})

	t.Run("bare installed current is skipped", func(t *testing.T) {
		proxyServer(t, "v0.7.0", nil)
		actions, examined, err := planGoSync([]string{"mvdan.cc/gofumpt"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, examined)
		assert.Empty(t, actions)
	})

	t.Run("proxy failure leaves tool untouched and is named", func(t *testing.T) {
		proxyServer(t, "", func(w http.ResponseWriter, r *http.Request, hits int) {
			http.Error(w, "boom", http.StatusInternalServerError)
		})
		actions, examined, err := planGoSync([]string{"mvdan.cc/gofumpt"}, []goTool{tool("mvdan.cc/gofumpt", "v0.7.0")}, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mvdan.cc/gofumpt")
		assert.Equal(t, 1, examined)
		assert.Empty(t, actions)
	})

	t.Run("cmd subpackage retries against owning module", func(t *testing.T) {
		proxyServer(t, "", func(w http.ResponseWriter, r *http.Request, hits int) {
			if strings.Contains(r.URL.Path, "/cmd/") {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"Version": "v0.99.0"}`)
		})
		tools := []goTool{{binary: "/go/bin/goimports", module: "golang.org/x/tools", version: "v0.24.0"}}
		actions, _, err := planGoSync([]string{"golang.org/x/tools/cmd/goimports"}, tools, false)
		// findGoTool matches the main package inside the module; the proxy
		// lookup falls back from .../cmd/goimports to the module itself.
		assert.NoError(t, err)
		assert.Len(t, actions, 1)
		assert.Equal(t, "upgrade", actions[0].verb)
		assert.Equal(t, "v0.99.0", actions[0].to)
	})
}

func TestGoProxyURL(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{"", "https://proxy.golang.org"},
		{"direct", "https://proxy.golang.org"},
		{"off", "https://proxy.golang.org"},
		{"https://goproxy.internal,https://proxy.golang.org", "https://goproxy.internal"},
		{"https://goproxy.internal/", "https://goproxy.internal"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("GOPROXY", tt.env)
			assert.Equal(t, tt.want, goProxyURL())
		})
	}
}

func TestEscapeGoModule(t *testing.T) {
	assert.Equal(t, "github.com/!foo/bar", escapeGoModule("github.com/Foo/bar"))
	assert.Equal(t, "golang.org/x/tools", escapeGoModule("golang.org/x/tools"))
	assert.Equal(t, "!a!b", escapeGoModule("AB"))
}
