package diff2

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_RealGit exercises the server against a real temp git repo.
// Skips if git is not installed.
func TestIntegration_RealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	tmp := t.TempDir()
	runCmd(t, tmp, "git", "init", "-q")
	runCmd(t, tmp, "git", "config", "user.email", "test@example.com")
	runCmd(t, tmp, "git", "config", "user.name", "test")
	writeFile(t, filepath.Join(tmp, "foo.go"), "package main\n")
	runCmd(t, tmp, "git", "add", "-A")
	runCmd(t, tmp, "git", "commit", "-q", "-m", "init")
	writeFile(t, filepath.Join(tmp, "foo.go"), "package main\n// new line\n")
	writeFile(t, filepath.Join(tmp, "new.go"), "// untracked\n")

	// Run tests with tmp as cwd.
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	h := NewServer(Deps{Git: execGit{}, Bd: newFakeBd().on("label", "[]").on("--version", "bd")})

	// /api/diff
	srv := httptest.NewServer(h)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/diff")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body=%s", res.StatusCode, string(b))
	}
	var body struct {
		Files []File `json:"files"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if len(body.Files) == 0 {
		t.Errorf("expected at least one changed file, got none")
	}
}

// TestIntegration_BeadCreate_WithRealBd uses a real temp bd DB.
func TestIntegration_BeadCreate_WithRealBd(t *testing.T) {
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("bd not installed")
	}

	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	// bd init in temp dir
	if out, err := exec.Command("bd", "init", "--prefix", "test").CombinedOutput(); err != nil {
		t.Fatalf("bd init: %v: %s", err, out)
	}

	h := NewServer(Deps{Git: newFakeGit().on("rev-parse --is-inside-work-tree", "true\n"), Bd: execBd{}})
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := strings.NewReader(`{"title":"integ test","type":"task","priority":"P2"}`)
	res, err := http.Post(srv.URL+"/api/bead", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 201 {
		t.Fatalf("status %d body=%s", res.StatusCode, string(b))
	}
	var br BeadResponse
	if err := json.Unmarshal(b, &br); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(br.ID, "test-") {
		t.Errorf("unexpected id %q", br.ID)
	}
}

// helpers -----------------------------------------------------------------

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
