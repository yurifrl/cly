package diff2

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestDeps() Deps {
	g := newFakeGit().
		on("rev-parse --is-inside-work-tree", "true\n").
		on("diff HEAD --name-status -z", "M\tfoo.go\x00").
		on("diff HEAD --numstat", "3\t1\tfoo.go\n").
		on("ls-files --others --exclude-standard -z", "").
		on("diff HEAD -- foo.go", `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,2 +1,2 @@
 keep
-old
+new
`)
	b := newFakeBd().
		on("--version", "bd v1.0.0").
		on("label", `["backend","refactor"]`).
		on("create", `{"id":"bd-7"}`)
	return Deps{Git: g, Bd: b}
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestHandleHealth(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "GET", "/api/health", "")
	if rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	var out map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out["git"] || !out["bd"] || !out["beadsDb"] {
		t.Errorf("want all true, got %v", out)
	}
}

func TestHandleDiff(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "GET", "/api/diff", "")
	if rec.Code != 200 {
		t.Fatalf("code %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Files []File `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Files) != 1 || out.Files[0].Path != "foo.go" {
		t.Errorf("files: %+v", out.Files)
	}
}

func TestHandleDiffFile(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "GET", "/api/diff/file?path=foo.go", "")
	if rec.Code != 200 {
		t.Fatalf("code %d body=%s", rec.Code, rec.Body.String())
	}
	var out FileDiff
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Hunks) != 1 {
		t.Errorf("hunks: %+v", out.Hunks)
	}
}

func TestHandleDiffFile_MissingPath(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "GET", "/api/diff/file", "")
	if rec.Code != 400 {
		t.Errorf("code %d", rec.Code)
	}
}

func TestHandleLabels(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "GET", "/api/labels", "")
	if rec.Code != 200 {
		t.Fatalf("code %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Labels []string `json:"labels"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Labels) != 2 {
		t.Errorf("labels: %+v", out.Labels)
	}
}

func TestHandleBead_Success(t *testing.T) {
	h := NewServer(newTestDeps())
	body := `{"title":"x","type":"task","priority":"P2","context":"foo.go","labels":["a"]}`
	rec := do(t, h, "POST", "/api/bead", body)
	if rec.Code != 201 {
		t.Fatalf("code %d body=%s", rec.Code, rec.Body.String())
	}
	var out BeadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != "bd-7" {
		t.Errorf("id: %s", out.ID)
	}
}

func TestHandleBead_MethodNotAllowed(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "GET", "/api/bead", "")
	if rec.Code != 405 {
		t.Errorf("code %d", rec.Code)
	}
}

func TestHandleBead_NoDB(t *testing.T) {
	d := newTestDeps()
	d.Bd = newFakeBd().onErr("create", "Error: no beads database found", errors.New("exit 1"))
	h := NewServer(d)
	body := `{"title":"x","type":"task","priority":"P2"}`
	rec := do(t, h, "POST", "/api/bead", body)
	if rec.Code != 409 {
		t.Errorf("code %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleBead_BadInput(t *testing.T) {
	h := NewServer(newTestDeps())
	body := `{"title":"","type":"task","priority":"P2"}`
	rec := do(t, h, "POST", "/api/bead", body)
	if rec.Code != 400 {
		t.Errorf("code %d", rec.Code)
	}
}

func TestHandleBead_BadJSON(t *testing.T) {
	h := NewServer(newTestDeps())
	rec := do(t, h, "POST", "/api/bead", "{not json")
	if rec.Code != 400 {
		t.Errorf("code %d", rec.Code)
	}
}

func TestListen(t *testing.T) {
	l, port, err := Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if port == 0 {
		t.Error("port should be assigned")
	}
	if !strings.HasPrefix(l.Addr().String(), "127.0.0.1:") {
		t.Errorf("addr: %s", l.Addr())
	}
}

var _ = bytes.NewBuffer
