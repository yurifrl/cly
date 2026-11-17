package bundle

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeTern puts a `tern` stand-in on PATH that manages $PLUGIN_DIR the
// way the real one lays it out: install copies a directory named after the
// source's last path segment (or $INSTALL_AS), link writes <basename>.path,
// remove/unlink delete them. Every call is appended to $CALL_LOG.
func writeFakeTern(t *testing.T) (pluginDir, callLog string) {
	t.Helper()
	bin := t.TempDir()
	pluginDir = filepath.Join(t.TempDir(), "plugins")
	callLog = filepath.Join(t.TempDir(), "calls")
	script := `#!/bin/sh
shift # plugin
echo "$*" >> "$CALL_LOG"
case "$1" in
dir) printf '{"dir":"%s"}\n' "$PLUGIN_DIR" ;;
install) id="${INSTALL_AS:-$(basename "$2")}"; [ -n "$FAIL_INSTALL" ] && { echo "clone failed" >&2; exit 1; }
  mkdir -p "$PLUGIN_DIR/$id" ;;
remove) rm -rf "$PLUGIN_DIR/$2" ;;
link) mkdir -p "$PLUGIN_DIR"; printf '%s\n' "$2" > "$PLUGIN_DIR/$(basename "$2").path" ;;
unlink) rm -f "$PLUGIN_DIR/$2.path" ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "tern"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PLUGIN_DIR", pluginDir)
	t.Setenv("CALL_LOG", callLog)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return pluginDir, callLog
}

func writeTernfile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Ternfile")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func calls(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if l != "" && !strings.HasPrefix(l, "dir") {
			out = append(out, l)
		}
	}
	return out
}

func mkdir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseTernfile(t *testing.T) {
	linkDir := t.TempDir()
	t.Setenv("TERN_TEST_DIR", linkDir)
	path := writeTernfile(t, `# plugins
chirp  github.com/Noctivoro/tern-chirp   # robot voice

statusline link:$TERN_TEST_DIR
board      link:~/nowhere?
`)
	entries, err := parseTernfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %+v, want 3", entries)
	}
	if e := entries[0]; e.id != "chirp" || e.source != "github.com/Noctivoro/tern-chirp" || e.link {
		t.Errorf("chirp = %+v", e)
	}
	if e := entries[1]; !e.link || e.optional || e.source != linkDir {
		t.Errorf("statusline = %+v, want link to %s", e, linkDir)
	}
	home, _ := os.UserHomeDir()
	if e := entries[2]; !e.link || !e.optional || e.source != filepath.Join(home, "nowhere") {
		t.Errorf("board = %+v, want optional link", e)
	}
}

func TestParseTernfileErrors(t *testing.T) {
	for name, content := range map[string]string{
		"missing source":   "chirp\n",
		"extra field":      "chirp a b\n",
		"duplicate id":     "a x\na y\n",
		"optional install": "a github.com/x/y?\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseTernfile(writeTernfile(t, content)); err == nil {
				t.Errorf("parseTernfile(%q) = nil error, want error", content)
			}
		})
	}
}

func TestPlanTern(t *testing.T) {
	linkDir := mkdir(t, filepath.Join(t.TempDir(), "sl"))
	otherDir := mkdir(t, filepath.Join(t.TempDir(), "other"))
	desired := []ternEntry{
		{id: "kept", source: "github.com/a/kept"},
		{id: "new", source: "github.com/a/new"},
		{id: "sl", source: linkDir, link: true},
		{id: "moved", source: linkDir, link: true},
		{id: "was-link", source: "github.com/a/was-link"},
		{id: "opt", source: "/nope", link: true, optional: true},
		{id: "req", source: "/nope", link: true},
	}
	installed := map[string]ternInstalled{
		"kept":     {},
		"sl":       {link: true, target: linkDir},
		"moved":    {link: true, target: otherDir},
		"was-link": {link: true, target: otherDir},
		"zzz":      {},
		"broken":   {link: true, target: "/gone", broken: true},
	}
	got := map[string]string{}
	var order []string
	for _, a := range planTern(desired, installed, false) {
		got[a.id] = a.verb
		order = append(order, a.id)
	}
	want := map[string]string{
		"kept": ternOK, "new": ternInstall, "sl": ternOK, "moved": ternReplace,
		"was-link": ternReplace, "opt": ternSkip, "req": ternMissing,
		"broken": ternRemove, "zzz": ternRemove,
	}
	for id, verb := range want {
		if got[id] != verb {
			t.Errorf("%s: verb = %q, want %q", id, got[id], verb)
		}
	}
	if strings.Join(order[len(order)-2:], ",") != "broken,zzz" {
		t.Errorf("removals should come last, sorted: %v", order)
	}

	for _, a := range planTern(desired[:1], installed, true) {
		if a.id == "kept" && a.verb != ternReinstall {
			t.Errorf("force: kept verb = %q, want reinstall", a.verb)
		}
	}
}

func TestTernSyncAndCleanup(t *testing.T) {
	withTempConfig(t, nil)
	pluginDir, log := writeFakeTern(t)
	linkDir := mkdir(t, filepath.Join(t.TempDir(), "statusline"))
	mkdir(t, filepath.Join(pluginDir, "chirp"))
	mkdir(t, filepath.Join(pluginDir, "stale"))
	if err := os.WriteFile(filepath.Join(pluginDir, "probe.path"), []byte("/gone\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tf := writeTernfile(t, "chirp github.com/x/chirp\njsonx github.com/x/sdk/jsonx\nstatusline link:"+linkDir+"\nboard link:/nope?\n")
	b := NewTernBundler()

	if err := b.Check(tf); err == nil {
		t.Error("Check before sync = nil, want changes needed")
	}
	if c := calls(t, log); len(c) != 0 {
		t.Errorf("Check must not change anything, ran %v", c)
	}

	if err := b.Sync(tf, false, false, false, false, false); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if got, want := strings.Join(calls(t, log), "|"), "install github.com/x/sdk/jsonx|link "+linkDir; got != want {
		t.Errorf("sync calls = %q, want %q", got, want)
	}
	if !isDir(filepath.Join(pluginDir, "stale")) {
		t.Error("Sync must leave unlisted plugins for Cleanup")
	}

	os.Remove(log)
	if err := b.Cleanup(tf, false, true); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if got, want := strings.Join(calls(t, log), "|"), "unlink probe|remove stale"; got != want {
		t.Errorf("cleanup calls = %q, want %q", got, want)
	}

	if err := b.Check(tf); err != nil {
		t.Errorf("Check after sync+cleanup = %v, want in sync", err)
	}
}

func TestTernSyncDetectsWrongID(t *testing.T) {
	withTempConfig(t, nil)
	writeFakeTern(t)
	t.Setenv("INSTALL_AS", "real-id")
	err := NewTernBundler().Sync(writeTernfile(t, "guess github.com/x/thing\n"), false, false, false, false, false)
	if err == nil {
		t.Fatal("Sync = nil, want failure for mismatched id")
	}
}

func TestTernSyncReportsInstallFailure(t *testing.T) {
	withTempConfig(t, nil)
	writeFakeTern(t)
	t.Setenv("FAIL_INSTALL", "1")
	err := NewTernBundler().Sync(writeTernfile(t, "chirp github.com/x/chirp\n"), false, false, false, false, false)
	if err == nil || !strings.Contains(err.Error(), "chirp") {
		t.Fatalf("Sync err = %v, want failure naming chirp", err)
	}
}

func TestTernSyncFailsOnMissingRequiredLink(t *testing.T) {
	withTempConfig(t, nil)
	writeFakeTern(t)
	if err := NewTernBundler().Sync(writeTernfile(t, "sl link:/definitely/not/here\n"), false, false, false, false, false); err == nil {
		t.Fatal("Sync = nil, want failure for missing required link target")
	}
}

// writeFakeGit puts a `git` stand-in on PATH whose `clone ... --branch TAG
// URL DEST` creates DEST/$CLONE_SUBPATH and logs "clone URL TAG".
func writeFakeGit(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/bin/sh
echo "clone $7 $6" >> "$CALL_LOG"
[ -n "$FAIL_CLONE" ] && { echo "fatal: Remote branch $6 not found" >&2; exit 128; }
mkdir -p "$8/$CLONE_SUBPATH"
`
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// fakeReleases serves /repos/OWNER/REPO/releases/latest from tags; repos
// not in the map 404 like a repo without releases.
func fakeReleases(t *testing.T, tags map[string]string) *int {
	t.Helper()
	hits := new(int)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		repo := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/repos/"), "/releases/latest")
		tag, ok := tags[repo]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"tag_name":%q}`, tag)
	}))
	t.Cleanup(srv.Close)
	old := ternGitHubAPI
	ternGitHubAPI = srv.URL
	t.Cleanup(func() { ternGitHubAPI = old })
	return hits
}

func TestParseTernfileRefs(t *testing.T) {
	entries, err := parseTernfile(writeTernfile(t, `a github.com/o/r@v1.2.0
b github.com/o/mono/plugins/b@go/tern/v0.2.0
c https://github.com/o/c.git@latest
d github.com/o/d
e git@github.com:o/e.git
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ repo, subpath, ref string }{
		{"o/r", "", "v1.2.0"},
		{"o/mono", "plugins/b", "go/tern/v0.2.0"},
		{"o/c", "", "latest"},
		{"o/d", "", ""},
		{"", "", ""},
	}
	for i, w := range want {
		e := entries[i]
		if e.repo != w.repo || e.subpath != w.subpath || e.ref != w.ref {
			t.Errorf("%s = repo %q subpath %q ref %q, want %+v", e.id, e.repo, e.subpath, e.ref, w)
		}
	}

	for name, content := range map[string]string{
		"empty tag":        "a github.com/o/r@\n",
		"no repo":          "a github.com/o@v1\n",
		"tag on local dir": "a ./plugins/a@v1\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseTernfile(writeTernfile(t, content)); err == nil {
				t.Errorf("parseTernfile(%q) = nil error, want error", content)
			}
		})
	}
}

func TestPlanTernRefs(t *testing.T) {
	desired := []ternEntry{
		{id: "same", tag: "v1.0.0"},
		{id: "up", tag: "v1.2.0"},
		{id: "down", tag: "v0.9.0"},
		{id: "untracked", tag: "v1.0.0"},
		{id: "unpin"},
		{id: "fresh", tag: "v1.0.0"},
		{id: "broken", err: fmt.Errorf("no releases")},
	}
	installed := map[string]ternInstalled{
		"same":      {ref: "v1.0.0"},
		"up":        {ref: "v1.0.0"},
		"down":      {ref: "v1.0.0"},
		"untracked": {},
		"unpin":     {ref: "v1.0.0"},
		"broken":    {},
	}
	want := map[string]string{
		"same": ternOK, "up": ternUpgrade, "down": ternDowngrade, "untracked": ternPin,
		"unpin": ternReinstall, "fresh": ternInstall, "broken": ternError,
	}
	for _, a := range planTern(desired, installed, false) {
		if a.verb != want[a.id] {
			t.Errorf("%s: verb = %q (%s), want %q", a.id, a.verb, a.note, want[a.id])
		}
	}
	for _, a := range planTern(desired[:1], installed, true) {
		if a.id == "same" && a.verb != ternReinstall {
			t.Errorf("force on pinned: verb = %q, want reinstall", a.verb)
		}
	}
}

func TestResolveTernLatest(t *testing.T) {
	hits := fakeReleases(t, map[string]string{"o/r": "v2.0.0"})
	entries := []ternEntry{
		{id: "a", repo: "o/r", ref: "latest"},
		{id: "b", repo: "o/r", ref: "latest"},
		{id: "c", repo: "o/none", ref: "latest"},
		{id: "d", repo: "o/r", ref: "v1.0.0"},
	}
	resolveTernRefs(entries)
	if entries[0].tag != "v2.0.0" || entries[1].tag != "v2.0.0" {
		t.Errorf("latest tags = %q, %q, want v2.0.0", entries[0].tag, entries[1].tag)
	}
	if entries[2].err == nil || !strings.Contains(entries[2].err.Error(), "no GitHub releases") {
		t.Errorf("repo without releases: err = %v", entries[2].err)
	}
	if entries[3].tag != "v1.0.0" {
		t.Errorf("explicit tag = %q, want v1.0.0", entries[3].tag)
	}
	if *hits != 2 {
		t.Errorf("API hits = %d, want 2 (one per repo)", *hits)
	}
}

func TestTernSyncPinnedLifecycle(t *testing.T) {
	withTempConfig(t, nil)
	pluginDir, log := writeFakeTern(t)
	writeFakeGit(t)
	fakeReleases(t, map[string]string{"o/sdk": "v0.3.0"})
	t.Setenv("CLONE_SUBPATH", "plugins/jsonx")
	b := NewTernBundler()
	marker := func() string {
		data, _ := os.ReadFile(filepath.Join(pluginDir, "jsonx", ternRefMarker))
		return strings.TrimSpace(string(data))
	}
	sync := func(content string) []string {
		t.Helper()
		os.Remove(log)
		if err := b.Sync(writeTernfile(t, content), false, false, false, false, false); err != nil {
			t.Fatalf("Sync(%q): %v", content, err)
		}
		return calls(t, log)
	}

	got := sync("jsonx github.com/o/sdk/plugins/jsonx@v0.2.0\n")
	if len(got) != 2 || got[0] != "clone https://github.com/o/sdk.git v0.2.0" ||
		!strings.HasPrefix(got[1], "install ") || !strings.HasSuffix(got[1], "/repo/plugins/jsonx") {
		t.Errorf("pinned install calls = %v", got)
	}
	if marker() != "v0.2.0" {
		t.Errorf("marker = %q, want v0.2.0", marker())
	}

	if got := sync("jsonx github.com/o/sdk/plugins/jsonx@v0.2.0\n"); len(got) != 0 {
		t.Errorf("same tag should be a no-op, ran %v", got)
	}

	got = sync("jsonx github.com/o/sdk/plugins/jsonx@latest\n")
	if len(got) != 2 || got[0] != "clone https://github.com/o/sdk.git v0.3.0" || !strings.HasSuffix(got[1], " --force") {
		t.Errorf("@latest upgrade calls = %v", got)
	}
	if marker() != "v0.3.0" {
		t.Errorf("marker = %q, want v0.3.0", marker())
	}

	got = sync("jsonx github.com/o/sdk/plugins/jsonx\n")
	if len(got) != 1 || got[0] != "install github.com/o/sdk/plugins/jsonx --force" {
		t.Errorf("unpin calls = %v", got)
	}
	if marker() != "" {
		t.Errorf("marker after unpin = %q, want none", marker())
	}
}

func TestTernSyncCloneFailure(t *testing.T) {
	withTempConfig(t, nil)
	_, log := writeFakeTern(t)
	writeFakeGit(t)
	t.Setenv("FAIL_CLONE", "1")
	err := NewTernBundler().Sync(writeTernfile(t, "x github.com/o/x@v9.9.9\n"), false, false, false, false, false)
	if err == nil {
		t.Fatal("Sync = nil, want failure when the tag cannot be cloned")
	}
	for _, c := range calls(t, log) {
		if strings.HasPrefix(c, "install") {
			t.Errorf("must not install after a failed clone, ran %q", c)
		}
	}
}
