package diff2

import (
	"fmt"
	"strings"
	"testing"
)

// fakeGit records calls and returns scripted outputs.
type fakeGit struct {
	outs  map[string][]byte
	errs  map[string]error
	calls [][]string
}

func newFakeGit() *fakeGit {
	return &fakeGit{outs: map[string][]byte{}, errs: map[string]error{}}
}

func (f *fakeGit) on(args string, out string) *fakeGit {
	f.outs[args] = []byte(out)
	return f
}

func (f *fakeGit) onErr(args string, err error) *fakeGit {
	f.errs[args] = err
	return f
}

func (f *fakeGit) Run(args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, args)
	if err, ok := f.errs[key]; ok {
		return nil, err
	}
	if out, ok := f.outs[key]; ok {
		return out, nil
	}
	return nil, fmt.Errorf("fakeGit: no stub for %q", key)
}

func TestParseNameStatusZ(t *testing.T) {
	// M\tfoo.go\0A\tbar.go\0D\tbaz.go\0
	in := "M\tfoo.go\x00A\tbar.go\x00D\tbaz.go\x00"
	got := parseNameStatusZ(in)
	if len(got) != 3 {
		t.Fatalf("want 3 files, got %d", len(got))
	}
	if got[0].Path != "foo.go" || got[0].Status != StatusModified {
		t.Errorf("file 0: %+v", got[0])
	}
	if got[1].Path != "bar.go" || got[1].Status != StatusAdded {
		t.Errorf("file 1: %+v", got[1])
	}
	if got[2].Path != "baz.go" || got[2].Status != StatusDeleted {
		t.Errorf("file 2: %+v", got[2])
	}
}

func TestParseNameStatusZ_Rename(t *testing.T) {
	// R100\told.go\0new.go\0
	in := "R100\told.go\x00new.go\x00"
	got := parseNameStatusZ(in)
	if len(got) != 1 {
		t.Fatalf("want 1 file, got %d", len(got))
	}
	if got[0].Status != StatusRenamed {
		t.Errorf("want renamed, got %s", got[0].Status)
	}
	if got[0].OldPath != "old.go" || got[0].Path != "new.go" {
		t.Errorf("rename paths: %+v", got[0])
	}
}

func TestApplyNumstat(t *testing.T) {
	files := []*File{{Path: "foo.go"}, {Path: "bar.png"}}
	in := "3\t1\tfoo.go\n-\t-\tbar.png\n"
	applyNumstat(files, in)
	if files[0].Additions != 3 || files[0].Deletions != 1 {
		t.Errorf("foo: %+v", files[0])
	}
	if !files[1].Binary {
		t.Errorf("bar should be binary")
	}
}

func TestIsBinaryDiff(t *testing.T) {
	if !isBinaryDiff("diff --git a/x b/x\nBinary files a/x and b/x differ\n") {
		t.Error("want binary detected")
	}
	if isBinaryDiff("diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n") {
		t.Error("text diff should not be binary")
	}
}

func TestParseHunks_Simple(t *testing.T) {
	raw := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,3 @@
 keep
-remove
+add
`
	hunks := parseHunks(raw)
	if len(hunks) != 1 {
		t.Fatalf("want 1 hunk, got %d", len(hunks))
	}
	h := hunks[0]
	if h.Header != "@@ -1,3 +1,3 @@" {
		t.Errorf("header: %q", h.Header)
	}
	if len(h.Lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(h.Lines))
	}
	want := []struct {
		kind string
		text string
	}{
		{"context", "keep"},
		{"del", "remove"},
		{"add", "add"},
	}
	for i, w := range want {
		if h.Lines[i].Kind != w.kind || h.Lines[i].Text != w.text {
			t.Errorf("line %d: got %+v want %+v", i, h.Lines[i], w)
		}
	}
}

func TestParseHunkHeader(t *testing.T) {
	o, n := parseHunkHeader("@@ -42,7 +42,12 @@ func foo() {")
	if o != 42 || n != 42 {
		t.Errorf("got old=%d new=%d want 42/42", o, n)
	}
	o, n = parseHunkHeader("@@ -1 +1 @@")
	if o != 1 || n != 1 {
		t.Errorf("single-line: got old=%d new=%d", o, n)
	}
}

func TestListChangedFiles_Integration(t *testing.T) {
	g := newFakeGit().
		on("diff HEAD --name-status -z", "M\tfoo.go\x00A\tbar.go\x00").
		on("diff HEAD --numstat", "3\t1\tfoo.go\n10\t0\tbar.go\n").
		on("ls-files --others --exclude-standard -z", "untracked.txt\x00")

	files, err := ListChangedFiles(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("want 3, got %d: %+v", len(files), files)
	}
	// find foo.go
	var foo *File
	for i := range files {
		if files[i].Path == "foo.go" {
			foo = &files[i]
		}
	}
	if foo == nil || foo.Additions != 3 || foo.Deletions != 1 {
		t.Errorf("foo: %+v", foo)
	}
}

func TestDiffFile_Binary(t *testing.T) {
	g := newFakeGit().on(
		"diff HEAD -- image.png",
		"diff --git a/image.png b/image.png\nBinary files a/image.png and b/image.png differ\n",
	)
	fd, err := DiffFile(g, "image.png")
	if err != nil {
		t.Fatal(err)
	}
	if !fd.Binary {
		t.Error("want binary=true")
	}
	if len(fd.Hunks) != 0 {
		t.Error("binary should have no hunks")
	}
}

func TestIsRepo(t *testing.T) {
	ok := newFakeGit().on("rev-parse --is-inside-work-tree", "true\n")
	if err := IsRepo(ok); err != nil {
		t.Errorf("want ok, got %v", err)
	}

	bad := newFakeGit().onErr("rev-parse --is-inside-work-tree", fmt.Errorf("boom"))
	if err := IsRepo(bad); err != ErrNotRepo {
		t.Errorf("want ErrNotRepo, got %v", err)
	}
}
