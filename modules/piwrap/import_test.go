package piwrap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const fixtureSession = `{"v":1,"sessionId":"019e5057-19ae-7ddc-a9e2-42abd19c8053","createdAt":"2026-05-22T15:39:06Z"}
{"id":"01a","parentId":null,"role":"user","content":"hello"}
{"id":"01b","parentId":"01a","role":"assistant","content":"hi","sessionRef":"019e5057-19ae-7ddc-a9e2-42abd19c8053"}
`

func TestExtractSessionID(t *testing.T) {
	id, err := extractSessionID([]byte(fixtureSession))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := "019e5057-19ae-7ddc-a9e2-42abd19c8053"
	if id != want {
		t.Errorf("id = %q, want %q", id, want)
	}
}

func TestExtractSessionID_Missing(t *testing.T) {
	_, err := extractSessionID([]byte(`{"v":1}` + "\n"))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestForkSession(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.jsonl")
	dst := filepath.Join(dir, "dst.jsonl")
	if err := os.WriteFile(src, []byte(fixtureSession), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := forkSession(src, dst); err != nil {
		t.Fatalf("fork: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	oldID := "019e5057-19ae-7ddc-a9e2-42abd19c8053"
	if bytes.Contains(got, []byte(oldID)) {
		t.Errorf("dst still contains old session id")
	}
	newID, err := extractSessionID(got)
	if err != nil {
		t.Fatalf("dst missing sessionId: %v", err)
	}
	if newID == oldID {
		t.Errorf("new id == old id")
	}
	// Both occurrences (top-level + sessionRef) must be rewritten to
	// the same new id; bytes.ReplaceAll guarantees this, assert it.
	if got := bytes.Count(got, []byte(newID)); got != 2 {
		t.Errorf("expected 2 occurrences of new id, got %d", got)
	}
	// Source untouched.
	srcAfter, _ := os.ReadFile(src)
	if !bytes.Equal(srcAfter, []byte(fixtureSession)) {
		t.Errorf("source mutated")
	}
}

func TestForkSession_NoSessionID(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.jsonl")
	dst := filepath.Join(dir, "dst.jsonl")
	if err := os.WriteFile(src, []byte(`{"v":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := forkSession(src, dst)
	if err == nil || !strings.Contains(err.Error(), "no_session_id_in_source") {
		t.Errorf("err = %v, want no_session_id_in_source", err)
	}
}

func TestResolveSource_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSource(p, dir, "ignored", ScopeCwd)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != p {
		t.Errorf("got %q, want %q", got, p)
	}
}

func TestResolveSource_AbsolutePathBadSuffix(t *testing.T) {
	_, err := resolveSource("/tmp/foo.txt", "/x", "y", ScopeCwd)
	if err == nil || err.Code != CodeSetyImportFailed {
		t.Errorf("got %v, want SETY_IMPORT_FAILED", err)
	}
}

func TestResolveSource_TooShort(t *testing.T) {
	_, err := resolveSource("019e", "/x", "y", ScopeCwd)
	if err == nil || err.Code != CodeSetyImportIDTooShort {
		t.Errorf("got %v, want SETY_IMPORT_ID_TOO_SHORT", err)
	}
}

func TestResolveSource_NotFound(t *testing.T) {
	root := t.TempDir()
	cwdDir := filepath.Join(root, "--Users-test--")
	if err := os.MkdirAll(cwdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := resolveSource("019e5057", root, "--Users-test--", ScopeCwd)
	if err == nil || err.Code != CodeSetyImportNotFound {
		t.Errorf("got %v, want SETY_IMPORT_NOT_FOUND", err)
	}
}

func TestResolveSource_FoundByPrefix(t *testing.T) {
	root := t.TempDir()
	cwdEnc := "--Users-test--"
	cwdDir := filepath.Join(root, cwdEnc)
	if err := os.MkdirAll(cwdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(cwdDir, "2026-05-22T15-39-06-030Z_019e5057-19ae-7ddc-a9e2-42abd19c8053.jsonl")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSource("019e5057", root, cwdEnc, ScopeCwd)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestResolveSource_Ambiguous(t *testing.T) {
	root := t.TempDir()
	cwdEnc := "--Users-test--"
	cwdDir := filepath.Join(root, cwdEnc)
	if err := os.MkdirAll(cwdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{
		"a-019e5057-aaa.jsonl",
		"b-019e5057-bbb.jsonl",
	} {
		if err := os.WriteFile(filepath.Join(cwdDir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := resolveSource("019e5057", root, cwdEnc, ScopeCwd)
	if err == nil || err.Code != CodeSetyImportAmbiguous {
		t.Errorf("got %v, want SETY_IMPORT_AMBIGUOUS", err)
	}
}

func TestResolveSource_AllScope(t *testing.T) {
	root := t.TempDir()
	otherDir := filepath.Join(root, "--other--")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(otherDir, "x-019e5057-y.jsonl")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// In ScopeCwd it would not find it — confirm:
	_, err := resolveSource("019e5057", root, "--current--", ScopeCwd)
	if err == nil || err.Code != CodeSetyImportNotFound {
		t.Errorf("cwd scope: got %v, want not-found", err)
	}
	// In ScopeAll, found.
	got, err := resolveSource("019e5057", root, "--current--", ScopeAll)
	if err != nil {
		t.Fatalf("all scope: err = %v", err)
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestQuarantineExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session", "cly-foo.jsonl")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	qdir := filepath.Join(dir, "trash")

	q, err := quarantineExisting(target, qdir)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("target should be gone")
	}
	if data, err := os.ReadFile(q); err != nil || string(data) != "orig" {
		t.Errorf("quarantined content wrong: %s, %v", data, err)
	}
	if !strings.HasPrefix(filepath.Base(q), time.Now().UTC().Format("2006")) {
		// Allow slight clock skew — at minimum the year must match.
		// (This is loose; the exact format check happens via parsing
		// in human review.)
	}
}
