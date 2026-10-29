package envs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFishLiteral(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"simple", "'simple'"},
		{"has 'quote'", `'has \'quote\''`},
		{"", "''"},
	}
	for _, tc := range tests {
		if got := fishLiteral(tc.in); got != tc.want {
			t.Errorf("fishLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnvQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain-value", "plain-value"},
		{"has space", `'has space'`},
		{"it's", `'it'\''s'`},
		{"dollar$sign", `'dollar$sign'`},
		{"a=b", "a=b"},
	}
	for _, tc := range tests {
		if got := envQuote(tc.in); got != tc.want {
			t.Errorf("envQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidEnvLabel(t *testing.T) {
	valid := []string{"API_KEY", "x", "A1_b", "HOME_DIR_2"}
	invalid := []string{"", "username", "password", "notesPlain", "server", "port", "database",
		"1starts-digit", "has-dash", "has space", "UPPER lower"}
	for _, l := range valid {
		if !validEnvLabel(l) {
			t.Errorf("validEnvLabel(%q) = false, want true", l)
		}
	}
	for _, l := range invalid {
		if validEnvLabel(l) {
			t.Errorf("validEnvLabel(%q) = true, want false", l)
		}
	}
}

func TestFilterValidFields(t *testing.T) {
	fields := []Field{
		{Label: "API_KEY", Value: "v1", Section: "work"},
		{Label: "password", Value: "secret"},
		{Label: "notesPlain", Value: "junk"},
		{Label: "", Value: "empty"},
		{Label: "VALID", Value: "v2", Section: "personal"},
	}
	got := filterValidFields(fields)
	if len(got) != 2 {
		t.Fatalf("filterValidFields kept %d fields, want 2: %+v", len(got), got)
	}
	if got[0].Label != "API_KEY" || got[1].Label != "VALID" {
		t.Errorf("unexpected survivors: %+v", got)
	}
}

func TestDecodeFieldsSectionLowercase(t *testing.T) {
	parsed := item{
		Fields: []struct {
			Label   string `json:"label"`
			Value   string `json:"value"`
			Type    string `json:"type"`
			Section *struct {
				Label string `json:"label"`
			} `json:"section"`
		}{
			{Label: "KEY", Value: "v", Section: &struct {
				Label string `json:"label"`
			}{Label: "Work"}},
			{Label: "OTHER", Value: "v"},
		},
	}
	got := decodeFields(parsed)
	if len(got) != 2 {
		t.Fatalf("decodeFields len = %d, want 2", len(got))
	}
	if got[0].Section != "work" {
		t.Errorf("section = %q, want %q", got[0].Section, "work")
	}
	if got[1].Section != "" {
		t.Errorf("missing section should be empty, got %q", got[1].Section)
	}
}

func TestBlobRoundTrip(t *testing.T) {
	fields := []Field{
		{Label: "A", Value: "one", Section: "general"},
		{Label: "B", Value: "with\ttab and more", Section: "work"},
		{Label: "C", Value: "'quoted'", Section: ""},
	}

	encoded := encodeBlob(fields)
	got, err := decodeBlob(encoded)
	if err != nil {
		t.Fatalf("decodeBlob: %v", err)
	}
	if len(got) != len(fields) {
		t.Fatalf("round trip got %d fields, want %d", len(got), len(fields))
	}
	for i := range fields {
		if got[i] != fields[i] {
			t.Errorf("field %d = %+v, want %+v", i, got[i], fields[i])
		}
	}
}

func TestWriteOutput(t *testing.T) {
	fields := []Field{
		{Label: "TOKEN", Value: "abc", Section: "work"},
		{Label: "TOKEN", Value: "def", Section: ""},
	}

	// sections off, plain
	var buf bytes.Buffer
	writeOutput(&buf, fields, false, false)
	want := "TOKEN=abc\nTOKEN=def\n"
	if buf.String() != want {
		t.Errorf("plain output = %q, want %q", buf.String(), want)
	}

	// sections on, plain — prefixed AND unprefixed lines, as in the original
	buf.Reset()
	writeOutput(&buf, fields, false, true)
	want = "WORK_TOKEN=abc\nTOKEN=abc\nGENERAL_TOKEN=def\nTOKEN=def\n"
	if buf.String() != want {
		t.Errorf("sections output = %q, want %q", buf.String(), want)
	}

	// fish, sections off
	buf.Reset()
	writeOutput(&buf, fields[:1], true, false)
	want = "set -gx TOKEN 'abc'\n"
	if buf.String() != want {
		t.Errorf("fish output = %q, want %q", buf.String(), want)
	}
}

func newTestAgeStore(t *testing.T) *ageStore {
	t.Helper()
	root := t.TempDir()
	s := &ageStore{
		sshDir:   filepath.Join(root, "ssh"),
		keyPath:  filepath.Join(root, "config", "cly", "envs.age.key"),
		blobPath: filepath.Join(root, "state", "cly", "envs.age"),
	}
	if err := os.MkdirAll(s.sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAgeStoreRoundTripGeneratedIdentity(t *testing.T) {
	s := newTestAgeStore(t)
	fields := []Field{
		{Label: "API_KEY", Value: "s3cret-value", Section: "work"},
		{Label: "HOME", Value: "/home/yuri", Section: ""},
	}

	if err := s.Store(fields); err != nil {
		t.Fatalf("Store: %v", err)
	}
	// Cache file must be 0600 and the identity persisted next to it.
	if fi, err := os.Stat(s.blobPath); err != nil {
		t.Fatalf("stat cache: %v", err)
	} else if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("cache file mode = %o, want 600", perm)
	}
	if _, err := os.Stat(s.keyPath); err != nil {
		t.Errorf("generated identity missing: %v", err)
	}

	got, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != len(fields) {
		t.Fatalf("got %d fields, want %d", len(got), len(fields))
	}
	for i := range fields {
		if got[i] != fields[i] {
			t.Errorf("field %d = %+v, want %+v", i, got[i], fields[i])
		}
	}
}

func TestAgeStoreSSHRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	s := newTestAgeStore(t)
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "",
		"-f", filepath.Join(s.sshDir, "id_ed25519"), "-q",
		"-C", "envs-test").CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v: %s", err, out)
	}

	fields := []Field{{Label: "SSH_SECRET", Value: "via-ssh-recipient", Section: "personal"}}
	if err := s.Store(fields); err != nil {
		t.Fatalf("Store: %v", err)
	}
	// With an SSH key present, no generated age identity should exist.
	if _, err := os.Stat(s.keyPath); !os.IsNotExist(err) {
		t.Errorf("identity file should not be created when SSH recipient is used, err=%v", err)
	}

	got, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 1 || got[0] != fields[0] {
		t.Fatalf("got %+v, want %+v", got, fields)
	}
}

func TestAgeStoreReadWithoutIdentities(t *testing.T) {
	s := newTestAgeStore(t)
	fields := []Field{{Label: "A", Value: "one", Section: ""}}
	if err := s.Store(fields); err != nil {
		t.Fatalf("Store: %v", err)
	}
	// Remove every key so decryption is impossible.
	os.Remove(s.keyPath)
	if err := os.RemoveAll(s.sshDir); err != nil {
		t.Fatal(err)
	}
	_, err := s.Read()
	if err == nil {
		t.Fatal("Read should fail without any identity")
	}
	if !strings.Contains(err.Error(), "no usable age identity") {
		t.Errorf("error = %v, want identity hint", err)
	}
}

func TestCacheBackendOverride(t *testing.T) {
	// The override key is read through pkg/config; the unit here only pins
	// the OS default mapping. Override behavior is covered by the smoke run.
	if runtime.GOOS == "darwin" && cacheBackend() != "keychain" {
		t.Errorf("darwin default = %q, want keychain", cacheBackend())
	}
}

func TestAgeStorePubOnlySSHFallsBackToIdentity(t *testing.T) {
	dir := t.TempDir()
	// 1Password-agent style setup: public key present, private half absent.
	pub := filepath.Join(dir, "id_ed25519.pub")
	if err := os.WriteFile(pub, []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakefakefakefakefakefakefakefakefakefakefake fake@host\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &ageStore{
		sshDir:   dir,
		keyPath:  filepath.Join(dir, "envs.age.key"),
		blobPath: filepath.Join(dir, "envs.age"),
	}
	fields := []Field{{Label: "TOKEN", Value: "sekret", Section: "work"}}
	if err := s.Store(fields); err != nil {
		t.Fatalf("Store: %v", err)
	}
	// Identity must have been generated (no SSH private key to reuse).
	if _, err := os.Stat(s.keyPath); err != nil {
		t.Fatalf("generated identity missing: %v", err)
	}
	got, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 1 || got[0].Label != "TOKEN" || got[0].Value != "sekret" {
		t.Fatalf("round trip got %+v", got)
	}
}
