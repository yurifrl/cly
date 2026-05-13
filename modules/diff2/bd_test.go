package diff2

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// fakeBd is a scripted BdRunner.
type fakeBd struct {
	// keyed by first arg (e.g. "label", "create")
	outs   map[string]string
	stderr map[string]string
	errs   map[string]error
	calls  [][]string
}

func newFakeBd() *fakeBd {
	return &fakeBd{
		outs:   map[string]string{},
		stderr: map[string]string{},
		errs:   map[string]error{},
	}
}

func (f *fakeBd) on(firstArg, out string) *fakeBd {
	f.outs[firstArg] = out
	return f
}

func (f *fakeBd) onErr(firstArg string, stderr string, err error) *fakeBd {
	f.stderr[firstArg] = stderr
	f.errs[firstArg] = err
	return f
}

func (f *fakeBd) Run(args ...string) ([]byte, []byte, error) {
	f.calls = append(f.calls, args)
	if len(args) == 0 {
		return nil, nil, errors.New("no args")
	}
	k := args[0]
	if err, ok := f.errs[k]; ok {
		return nil, []byte(f.stderr[k]), err
	}
	return []byte(f.outs[k]), nil, nil
}

func TestListLabels_EmptyJSON(t *testing.T) {
	b := newFakeBd().on("label", "[]")
	got, err := ListLabels(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestListLabels_StringArray(t *testing.T) {
	b := newFakeBd().on("label", `["backend","refactor"]`)
	got, err := ListLabels(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "backend" || got[1] != "refactor" {
		t.Errorf("got %v", got)
	}
}

func TestListLabels_NoDBDegrades(t *testing.T) {
	b := newFakeBd().onErr("label", "Error: no beads database found\n", errors.New("exit 1"))
	got, err := ListLabels(b)
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestListLabels_BdMissingDegrades(t *testing.T) {
	b := newFakeBd().onErr("label", "", &exec.Error{Name: "bd", Err: exec.ErrNotFound})
	got, err := ListLabels(b)
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestCreateBead_Success(t *testing.T) {
	b := newFakeBd().on("create", `{"id":"bd-42","title":"x"}`)
	resp, err := CreateBead(b, BeadRequest{
		Title: "x", Type: "task", Priority: "P2",
		Context: "foo.go", Labels: []string{"a", "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "bd-42" {
		t.Errorf("id: %s", resp.ID)
	}
	// verify priority normalized and args built
	last := b.calls[len(b.calls)-1]
	joined := strings.Join(last, " ")
	if !strings.Contains(joined, "--priority 2") {
		t.Errorf("want '--priority 2' in %q", joined)
	}
	if !strings.Contains(joined, "--labels a,b") {
		t.Errorf("want '--labels a,b' in %q", joined)
	}
	if !strings.Contains(joined, "--context foo.go") {
		t.Errorf("want context in %q", joined)
	}
	if !strings.Contains(joined, "--json") {
		t.Errorf("want --json in %q", joined)
	}
}

func TestCreateBead_EmptyTitle(t *testing.T) {
	b := newFakeBd()
	_, err := CreateBead(b, BeadRequest{Title: "   "})
	if !errors.Is(err, ErrBdBadInput) {
		t.Errorf("want ErrBdBadInput, got %v", err)
	}
}

func TestCreateBead_NoDB(t *testing.T) {
	b := newFakeBd().onErr("create", "Error: no beads database found", errors.New("exit 1"))
	_, err := CreateBead(b, BeadRequest{Title: "x", Type: "task", Priority: "P2"})
	if !errors.Is(err, ErrBdNoDB) {
		t.Errorf("want ErrBdNoDB, got %v", err)
	}
}

func TestCreateBead_BdMissing(t *testing.T) {
	b := newFakeBd().onErr("create", "", &exec.Error{Name: "bd", Err: exec.ErrNotFound})
	_, err := CreateBead(b, BeadRequest{Title: "x", Type: "task", Priority: "P2"})
	if !errors.Is(err, ErrBdMissing) {
		t.Errorf("want ErrBdMissing, got %v", err)
	}
}

func TestNormalizePriority(t *testing.T) {
	cases := map[string]string{
		"P2": "2", "p3": "3", "0": "0", "": "2", "X": "2", "P0": "0",
	}
	for in, want := range cases {
		if got := normalizePriority(in); got != want {
			t.Errorf("normalize(%q)=%q want %q", in, got, want)
		}
	}
}
