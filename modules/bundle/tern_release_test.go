package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTernReleaseSync(t *testing.T) {
	pluginDir, log := writeFakeTern(t)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{"pkg/plugin/plugin.toml": "id=x", "pkg/bin/tool": "bin"} {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/releases/latest":
			fmt.Fprint(w, `{"tag_name":"v1.0.0"}`)
		case "/repos/o/r/releases/tags/v1.0.0":
			fmt.Fprintf(w, `{"assets":[{"name":"r_1_%s_%s.tar.gz","browser_download_url":"%s/dl"}]}`, runtime.GOOS, runtime.GOARCH, srv.URL)
		case "/dl":
			w.Write(buf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oldAPI, oldRoot := ternGitHubAPI, ternReleaseRoot
	ternGitHubAPI = srv.URL
	root := t.TempDir()
	ternReleaseRoot = func() string { return root }
	defer func() { ternGitHubAPI, ternReleaseRoot = oldAPI, oldRoot }()

	file := writeTernfile(t, "x  github:o/r/plugin +bin\n")
	b := NewTernBundler()
	if err := b.Sync(file, false, false, false, false, false); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "x", "v1.0.0")
	for _, f := range []string{"plugin/plugin.toml", "bin/tool"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	got := calls(t, log)
	want := "link " + filepath.Join(dir, "plugin")
	if len(got) != 1 || got[0] != want {
		t.Errorf("calls = %v, want [%s]", got, want)
	}
	// Re-sync once linked is a no-op (the fake names links by basename).
	os.Remove(filepath.Join(pluginDir, "plugin.path"))
	os.WriteFile(filepath.Join(pluginDir, "x.path"), []byte(filepath.Join(dir, "plugin")+"\n"), 0o644)
	if err := b.Check(file); err != nil {
		t.Errorf("Check after sync: %v", err)
	}
}
