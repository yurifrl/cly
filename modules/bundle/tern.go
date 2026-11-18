package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

// TernBundler keeps Tern's plugins directory in sync with a Ternfile.
//
// Ternfile format, one plugin per line:
//
//	<id>  <source>
//	<id>  github.com/OWNER/REPO[/PATH]@<tag|latest>
//	<id>  link:<dir>[?]
//	<id>  release:github.com/OWNER/REPO/SUBPATH[@<tag|latest>]
//	<id>  github:OWNER/REPO[/PATH][@<tag|latest|HEAD>]
//
// `github:` keeps a plugin current on its own: without @ it follows the
// newest GitHub release when the repo has releases, else the default
// branch's HEAD commit; @latest requires a release, @HEAD always tracks the
// branch, @<tag> pins. The installed tag or commit goes in .cly-ref.
//
// <source> is anything `tern plugin install` accepts (git URL,
// github.com/OWNER/REPO[/PATH], directory). A github.com source may pin a
// git tag with @<tag>, or follow the newest GitHub release with @latest;
// the installed tag is recorded in <plugin>/.cly-ref. `link:` loads the plugin in
// place via `tern plugin link`; a trailing `?` makes a link optional, so a
// missing directory is skipped instead of failing (machine-specific plugins).
// `release:` downloads a GitHub release tarball (plugin + binaries) and links
// SUBPATH from the unpacked copy; see tern_release.go.
// The id must match the plugin's id; it is how installed plugins are matched.
type TernBundler struct {
	name        string
	defaultFile string
}

// NewTernBundler creates a new TernBundler.
func NewTernBundler() *TernBundler {
	file := pkgconfig.GetString("modules.bundle.tern_file")
	if file == "" {
		file = "~/.config/cly/bundles/Ternfile"
	}
	return &TernBundler{name: "tern", defaultFile: file}
}

func (b *TernBundler) Name() string        { return b.name }
func (b *TernBundler) DefaultFile() string { return b.defaultFile }

func (b *TernBundler) CheckDeps() error {
	if !commandExists("tern") {
		return fmt.Errorf("tern not found. Install Tern and make sure `tern` is on PATH")
	}
	return nil
}

// ternEntry is one Ternfile line.
type ternEntry struct {
	id       string
	source   string // install source, or link directory (expanded) when link
	link     bool
	optional bool
	release  bool   // release: entry, fetched by cly and linked
	root     string // release: unpack directory for the resolved tag
	dir      string // release: directory linked into Tern (root/subpath)
	repo     string // OWNER/REPO, for github.com sources
	subpath  string // package directory inside the repo
	ref      string // tag or "latest" as written; empty = default branch
	tag      string // ref resolved to a concrete tag (set by load)
	github   bool   // github: entry, always tracked (release or branch HEAD)
	branch   bool   // track the default branch; want is its HEAD commit
	want     string // tag or commit expected in .cly-ref; empty = untracked
	err      error  // resolving ref failed
}

// ternRefMarker is written into a pinned install so later syncs know
// which tag is in place. A reinstall outside cly replaces the directory
// and drops it, which reads as "untracked" and re-pins.
const ternRefMarker = ".cly-ref"

// ternGitHubAPI is the GitHub API root used to resolve @latest; tests
// point it at an httptest server.
var ternGitHubAPI = "https://api.github.com"

// ternInstalled is one plugin present in Tern's plugins directory.
type ternInstalled struct {
	link   bool   // <id>.path file rather than an installed copy
	target string // link target
	broken bool   // link target is not a directory
	ref    string // tag recorded by a pinned install (copies only)
}

// parseTernfile reads a Ternfile. Blank lines and `#` comments (full-line
// or trailing) are ignored.
func parseTernfile(path string) ([]ternEntry, error) {
	lines, err := parseFile(path)
	if err != nil {
		return nil, err
	}
	var entries []ternEntry
	seen := map[string]bool{}
	for _, line := range lines {
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("Ternfile: %q: want '<id> <source>'", line)
		}
		e := ternEntry{id: fields[0], source: fields[1]}
		if seen[e.id] {
			return nil, fmt.Errorf("Ternfile: plugin %q listed twice", e.id)
		}
		seen[e.id] = true
		if rest, ok := strings.CutPrefix(e.source, "link:"); ok {
			e.link = true
			if strings.HasSuffix(rest, "?") {
				e.optional = true
				rest = strings.TrimSuffix(rest, "?")
			}
			e.source = filepath.Clean(expandPath(os.ExpandEnv(rest)))
		} else if rest, ok := strings.CutPrefix(e.source, "release:"); ok {
			e.release, e.source = true, rest
			if err := parseTernGitHub(&e); err != nil {
				return nil, fmt.Errorf("Ternfile: %s: %w", e.id, err)
			}
			if e.repo == "" || e.subpath == "" {
				return nil, fmt.Errorf("Ternfile: %s: release: wants github.com/OWNER/REPO/SUBPATH[@tag]", e.id)
			}
			if e.ref == "" {
				e.ref = "latest"
			}
		} else if rest, ok := strings.CutPrefix(e.source, "github:"); ok {
			e.github = true
			e.source = "github.com/" + rest
			if err := parseTernGitHub(&e); err != nil {
				return nil, fmt.Errorf("Ternfile: %s: %w", e.id, err)
			}
			e.source = "github:" + rest
		} else if strings.HasSuffix(e.source, "?") {
			return nil, fmt.Errorf("Ternfile: %s: '?' (optional) only applies to link: entries", e.id)
		} else if err := parseTernGitHub(&e); err != nil {
			return nil, fmt.Errorf("Ternfile: %s: %w", e.id, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// parseTernGitHub splits github.com/OWNER/REPO[/PATH][@REF] into its
// parts. Sources that are not on github.com pass through untouched; an
// @ref on them is an error since only GitHub tags/releases are resolved.
func parseTernGitHub(e *ternEntry) error {
	src := strings.TrimPrefix(strings.TrimPrefix(e.source, "https://"), "http://")
	rest, ok := strings.CutPrefix(src, "github.com/")
	if !ok {
		if strings.Contains(e.source, "@") && !strings.Contains(e.source, "://") && !strings.HasPrefix(e.source, "git@") {
			return fmt.Errorf("@<tag> is only supported on github.com/OWNER/REPO sources")
		}
		return nil
	}
	path, ref, hasRef := strings.Cut(rest, "@")
	if hasRef && ref == "" {
		return fmt.Errorf("empty tag after @")
	}
	parts := strings.SplitN(strings.Trim(path, "/"), "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("want github.com/OWNER/REPO[/PATH], got %q", e.source)
	}
	e.repo = parts[0] + "/" + strings.TrimSuffix(parts[1], ".git")
	if len(parts) == 3 {
		e.subpath = parts[2]
	}
	e.ref = ref
	return nil
}

// displaySource renders an entry's source as written in the Ternfile,
// adding the resolved tag for @latest.
func (e ternEntry) displaySource() string {
	switch {
	case e.link:
		return "link:" + e.source
	case e.release && e.tag != "" && e.ref == "latest":
		return "release:" + e.source + " -> " + e.tag
	case e.release:
		return "release:" + e.source
	case e.github && e.branch && e.want != "":
		return e.source + " -> HEAD " + shortRef(e.want)
	case e.github && e.tag != "" && e.ref != e.tag:
		return e.source + " -> " + e.tag
	case e.ref == "latest" && e.tag != "":
		return e.source + " -> " + e.tag
	}
	return e.source
}

// errTernNoReleases marks a repo without GitHub releases; github: entries
// fall back to the default branch on it.
var errTernNoReleases = errors.New("no GitHub releases")

// isCommit reports whether ref is a full git commit hash.
func isCommit(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, c := range ref {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

// shortRef abbreviates commit hashes for display; tags pass through.
func shortRef(ref string) string {
	if isCommit(ref) {
		return ref[:7]
	}
	return ref
}

// resolveTernHead returns the commit at the tip of repo's default branch.
func resolveTernHead(repo string) (string, error) {
	out, err := exec.Command("git", "ls-remote", "https://github.com/"+repo+".git", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s: %w", repo, err)
	}
	sha, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
	if !isCommit(sha) {
		return "", fmt.Errorf("git ls-remote %s: no HEAD commit", repo)
	}
	return sha, nil
}

// resolveTernLatest returns the tag of repo's latest GitHub release.
func resolveTernLatest(repo string) (string, error) {
	req, err := http.NewRequest("GET", ternGitHubAPI+"/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	setGitHubAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve @latest for %s: %w", repo, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%s has %w; pin a tag with @<tag> instead", repo, errTernNoReleases)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve @latest for %s: HTTP %d", repo, resp.StatusCode)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil || rel.TagName == "" {
		return "", fmt.Errorf("resolve @latest for %s: no tag_name in response", repo)
	}
	return rel.TagName, nil
}

// ternPluginDir asks tern where its plugins directory is.
func ternPluginDir() (string, error) {
	out, err := exec.Command("tern", "plugin", "dir", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("tern plugin dir: %w", err)
	}
	var res struct {
		Dir string `json:"dir"`
	}
	if err := json.Unmarshal(out, &res); err != nil || res.Dir == "" {
		return "", fmt.Errorf("tern plugin dir: unexpected output %q", strings.TrimSpace(string(out)))
	}
	return res.Dir, nil
}

// listTernPlugins reads the plugins directory directly: a directory is an
// installed copy, an <id>.path file names a linked plugin's directory.
// Reading the directory works whether or not a Tern daemon is running.
func listTernPlugins(dir string) (map[string]ternInstalled, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]ternInstalled{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}
	installed := map[string]ternInstalled{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		if id, ok := strings.CutSuffix(name, ".path"); ok && !e.IsDir() {
			data, err := os.ReadFile(full)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", name, err)
			}
			target := filepath.Clean(strings.TrimSpace(string(data)))
			installed[id] = ternInstalled{link: true, target: target, broken: !isDir(target)}
			continue
		}
		if isDir(full) {
			ref, _ := os.ReadFile(filepath.Join(full, ternRefMarker))
			installed[name] = ternInstalled{ref: strings.TrimSpace(string(ref))}
		}
	}
	return installed, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Plan verbs.
const (
	ternOK        = "ok"
	ternInstall   = "install"
	ternReinstall = "reinstall"
	ternLink      = "link"
	ternReplace   = "replace"   // swap an installed copy/link for the Ternfile's form
	ternUpgrade   = "upgrade"   // pinned tag moves forward
	ternDowngrade = "downgrade" // pinned tag moves back
	ternPin       = "pin"       // untracked copy gets the Ternfile's tag
	ternUpdate    = "update"    // tracked branch moved to a new commit
	ternFetch     = "fetch"     // download a release and link it
	ternSkip      = "skip"
	ternMissing   = "missing" // required link target does not exist
	ternError     = "error"   // the entry's tag could not be resolved
	ternRemove    = "remove"
)

type ternAction struct {
	verb  string
	entry ternEntry     // desired entry (zero for removals)
	have  ternInstalled // current state, when present
	id    string
	note  string
}

func (a ternAction) changes() bool {
	return a.verb != ternOK && a.verb != ternSkip
}

// planTern compares the Ternfile with the plugins directory. Desired
// entries come first in Ternfile order, then removals sorted by id.
func planTern(desired []ternEntry, installed map[string]ternInstalled, force bool) []ternAction {
	var actions []ternAction
	want := map[string]bool{}
	for _, e := range desired {
		want[e.id] = true
		have, present := installed[e.id]
		a := ternAction{entry: e, have: have, id: e.id}
		switch {
		case e.err != nil:
			a.verb, a.note = ternError, e.err.Error()
		case e.release && present && have.link && have.target == e.dir && isDir(e.dir) && !force:
			a.verb = ternOK
		case e.release && present:
			a.verb, a.note = ternFetch, "currently "+describeTern(have)
		case e.release:
			a.verb = ternFetch
		case e.link && !isDir(e.source):
			if e.optional {
				a.verb, a.note = ternSkip, "link target absent (optional)"
			} else {
				a.verb, a.note = ternMissing, "link target does not exist"
			}
		case e.link && present && have.link && have.target == e.source:
			a.verb = ternOK
		case e.link && present:
			a.verb, a.note = ternReplace, "currently "+describeTern(have)
		case e.link:
			a.verb = ternLink
		case present && !have.link && force:
			a.verb = ternReinstall
		case present && !have.link && e.want != "":
			a.verb, a.note = refVerb(have.ref, e.want)
		case present && !have.link && have.ref != "":
			a.verb, a.note = ternReinstall, "unpin from "+have.ref
		case present && !have.link:
			a.verb = ternOK
		case present:
			a.verb, a.note = ternReplace, "currently "+describeTern(have)
		default:
			a.verb = ternInstall
		}
		actions = append(actions, a)
	}

	var extras []string
	for id := range installed {
		if !want[id] {
			extras = append(extras, id)
		}
	}
	sort.Strings(extras)
	for _, id := range extras {
		have := installed[id]
		actions = append(actions, ternAction{verb: ternRemove, id: id, have: have, note: describeTern(have)})
	}
	return actions
}

// refVerb picks the action for an installed copy whose desired tag is
// want and recorded tag is have.
func refVerb(have, want string) (verb, note string) {
	switch {
	case have == want:
		return ternOK, ""
	case have == "":
		return ternPin, "untracked -> " + shortRef(want)
	case isCommit(have) || isCommit(want):
		return ternUpdate, shortRef(have) + " -> " + shortRef(want)
	case compareVersions(have, want) > 0:
		return ternDowngrade, have + " -> " + want
	case compareVersions(have, want) < 0:
		return ternUpgrade, have + " -> " + want
	}
	return ternReinstall, have + " -> " + want
}

func describeTern(t ternInstalled) string {
	switch {
	case t.link && t.broken:
		return "broken link -> " + t.target
	case t.link:
		return "linked -> " + t.target
	case t.ref != "":
		return "installed copy @" + shortRef(t.ref)
	default:
		return "installed copy"
	}
}

// ternLine renders one plan row: a status glyph, the id and its detail.
func ternLine(a ternAction) string {
	glyph, st := "✓", style.GreenStyle
	switch a.verb {
	case ternInstall, ternLink, ternFetch:
		glyph = "+"
	case ternUpgrade, ternUpdate:
		glyph = "↑"
	case ternDowngrade:
		glyph, st = "↓", style.YellowStyle
	case ternReinstall, ternReplace, ternPin:
		glyph, st = "↻", style.YellowStyle
	case ternSkip:
		glyph, st = "·", style.SubtleStyle
	case ternMissing, ternError:
		glyph, st = "✗", style.RedStyle
	case ternRemove:
		glyph, st = "-", style.RedStyle
	}
	detail := a.entry.displaySource()
	if a.verb == ternRemove {
		detail = ""
	}
	if a.note != "" {
		if detail != "" {
			detail += "  "
		}
		detail += "(" + a.note + ")"
	}
	return fmt.Sprintf("  %s %-10s %-22s %s", st.Render(glyph), a.verb, a.id, style.SubtleStyle.Render(detail))
}

func (b *TernBundler) load(bundleFile string, force bool) (string, []ternAction, error) {
	desired, err := parseTernfile(bundleFile)
	if err != nil {
		return "", nil, err
	}
	resolveTernRefs(desired)
	dir, err := ternPluginDir()
	if err != nil {
		return "", nil, err
	}
	installed, err := listTernPlugins(dir)
	if err != nil {
		return "", nil, err
	}
	return dir, planTern(desired, installed, force), nil
}

// resolveTernRefs fills each entry's tag: @<tag> as written, @latest via
// the GitHub releases API (once per repo). Failures stay on the entry and
// surface as an error row instead of aborting the whole plan. github:
// entries are resolved concurrently by resolveTernGitHub.
func resolveTernRefs(entries []ternEntry) {
	resolveTernGitHub(entries)
	latest := map[string]string{}
	failed := map[string]error{}
	for i := range entries {
		e := &entries[i]
		if e.github {
			continue
		}
		if e.ref != "latest" {
			e.tag = e.ref
			e.want = e.tag
			continue
		}
		if err, ok := failed[e.repo]; ok {
			e.err = err
			continue
		}
		tag, ok := latest[e.repo]
		if !ok {
			var err error
			if tag, err = resolveTernLatest(e.repo); err != nil {
				failed[e.repo], e.err = err, err
				continue
			}
			latest[e.repo] = tag
		}
		e.tag = tag
		e.want = tag
	}
	for i := range entries {
		entries[i].setReleasePaths()
	}
}

// resolveTernGitHub resolves github: entries, at most 8 lookups at a time:
// a release tag, else (no releases, or @HEAD) the default branch's commit.
func resolveTernGitHub(entries []ternEntry) {
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i := range entries {
		e := &entries[i]
		if !e.github {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			switch e.ref {
			case "HEAD":
			case "":
				tag, err := resolveTernLatest(e.repo)
				if err == nil {
					e.tag, e.want = tag, tag
					return
				}
				if !errors.Is(err, errTernNoReleases) {
					e.err = err
					return
				}
			case "latest":
				e.tag, e.err = resolveTernLatest(e.repo)
				e.want = e.tag
				return
			default:
				e.tag, e.want = e.ref, e.ref
				return
			}
			e.branch = true
			e.want, e.err = resolveTernHead(e.repo)
		}()
	}
	wg.Wait()
}

// runTern runs `tern plugin ARGS...`, streaming output when verbose and
// folding captured output into the error otherwise.
func runTern(verbose bool, args ...string) error {
	cmd := exec.Command("tern", append([]string{"plugin"}, args...)...)
	if verbose {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
	}
	return err
}

// uninstallTern removes whatever form the plugin currently has.
func uninstallTern(id string, have ternInstalled, verbose bool) error {
	if have.link {
		return runTern(verbose, "unlink", id)
	}
	return runTern(verbose, "remove", id)
}

func (b *TernBundler) apply(dir string, a ternAction, verbose bool) error {
	switch a.verb {
	case ternInstall:
		return b.install(dir, a, false, verbose)
	case ternReinstall, ternUpgrade, ternDowngrade, ternPin, ternUpdate:
		return b.install(dir, a, true, verbose)
	case ternLink:
		return runTern(verbose, "link", a.entry.source)
	case ternFetch:
		if err := fetchTernRelease(a.entry); err != nil {
			return err
		}
		if a.have != (ternInstalled{}) {
			if err := uninstallTern(a.id, a.have, verbose); err != nil {
				return err
			}
		}
		return runTern(verbose, "link", a.entry.dir)
	case ternReplace:
		if err := uninstallTern(a.id, a.have, verbose); err != nil {
			return err
		}
		if a.entry.link {
			return runTern(verbose, "link", a.entry.source)
		}
		return b.install(dir, a, false, verbose)
	case ternRemove:
		return uninstallTern(a.id, a.have, verbose)
	case ternMissing:
		return fmt.Errorf("link target %s does not exist", a.entry.source)
	case ternError:
		return a.entry.err
	case ternOK, ternSkip:
		return nil
	}
	return fmt.Errorf("internal: no handler for %q", a.verb)
}

// install runs `tern plugin install` and verifies the plugin landed under
// the Ternfile's id; a mismatch means the Ternfile names the wrong id and
// would otherwise reinstall forever while cleanup removes the real one.
// Pinned entries are cloned at their tag first (tern installs only the
// default branch) and get a ref marker afterwards.
func (b *TernBundler) install(dir string, a ternAction, force bool, verbose bool) error {
	e := a.entry
	src := e.source
	installed := e.tag
	if e.tag != "" || e.branch {
		tmp, err := os.MkdirTemp("", "cly-tern-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		clone := filepath.Join(tmp, "repo")
		args := []string{"clone", "--quiet", "--depth", "1"}
		label := "HEAD"
		if e.tag != "" {
			args, label = append(args, "--branch", e.tag), e.tag
		}
		args = append(args, "https://github.com/"+e.repo+".git", clone)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("git clone %s@%s: %w: %s", e.repo, label, err, strings.TrimSpace(string(out)))
		}
		if e.branch {
			// Record the commit actually cloned; the branch may have moved
			// since planning.
			out, err := exec.Command("git", "-C", clone, "rev-parse", "HEAD").Output()
			if err != nil {
				return fmt.Errorf("git rev-parse %s: %w", e.repo, err)
			}
			installed = strings.TrimSpace(string(out))
		}
		src = filepath.Join(clone, filepath.FromSlash(e.subpath))
		if !isDir(src) {
			return fmt.Errorf("%s has no directory %q at %s", e.repo, e.subpath, label)
		}
	}
	before, _ := listTernPlugins(dir)
	args := []string{"install", src}
	if force {
		args = append(args, "--force")
	}
	if err := runTern(verbose, args...); err != nil {
		return err
	}
	after, err := listTernPlugins(dir)
	if err != nil {
		return err
	}
	if _, ok := after[a.id]; ok {
		marker := filepath.Join(dir, a.id, ternRefMarker)
		if installed == "" {
			if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("clear ref marker: %w", err)
			}
			return nil
		}
		if err := os.WriteFile(marker, []byte(installed+"\n"), 0o644); err != nil {
			return fmt.Errorf("record installed tag: %w", err)
		}
		return nil
	}
	for id := range after {
		if _, existed := before[id]; !existed {
			return fmt.Errorf("installed as %q, not %q: fix the id in the Ternfile", id, a.id)
		}
	}
	return fmt.Errorf("install finished but no plugin %q appeared in %s", a.id, dir)
}

// Sync installs and links everything in the Ternfile. Removing plugins
// that are not listed is Cleanup's job (run after sync by default).
func (b *TernBundler) Sync(bundleFile string, verbose bool, force bool, _ bool, _ bool, _ bool) error {
	dir, actions, err := b.load(bundleFile, force)
	if err != nil {
		return err
	}
	var failed []string
	changed, kept := 0, 0
	for _, a := range actions {
		if a.verb == ternRemove {
			continue
		}
		fmt.Println(ternLine(a))
		if !a.changes() {
			kept++
			continue
		}
		if err := b.apply(dir, a, verbose); err != nil {
			printRed(fmt.Sprintf("    %s: %v", a.id, err))
			failed = append(failed, a.id)
			continue
		}
		changed++
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to sync %d plugin(s): %v", len(failed), failed)
	}
	if changed == 0 {
		fmt.Println()
		printGreen(fmt.Sprintf("Nothing changed - %d plugins in place", kept))
	} else {
		fmt.Println()
		printGreen(fmt.Sprintf("Done! %d changed, %d already in place", changed, kept))
	}
	return nil
}

// Check prints the plan without touching anything and fails when the
// plugins directory differs from the Ternfile.
func (b *TernBundler) Check(bundleFile string) error {
	_, actions, err := b.load(bundleFile, false)
	if err != nil {
		return err
	}
	pending := 0
	for _, a := range actions {
		fmt.Println(ternLine(a))
		if a.changes() {
			pending++
		}
	}
	if pending == 0 {
		fmt.Println()
		printGreen("Everything is in sync")
		return nil
	}
	fmt.Println()
	printYellow(fmt.Sprintf("%d change(s) needed", pending))
	return fmt.Errorf("changes needed")
}

// Cleanup removes or unlinks plugins that the Ternfile does not list,
// including broken links.
func (b *TernBundler) Cleanup(bundleFile string, verbose bool, _ bool) error {
	dir, actions, err := b.load(bundleFile, false)
	if err != nil {
		return err
	}
	removed := 0
	for _, a := range actions {
		if a.verb != ternRemove {
			continue
		}
		fmt.Println(ternLine(a))
		if err := b.apply(dir, a, verbose); err != nil {
			printRed(fmt.Sprintf("    Warning: failed to remove %s: %v", a.id, err))
			continue
		}
		removed++
	}
	if removed == 0 {
		printGreen("Nothing to clean up")
	} else {
		fmt.Println()
		printGreen(fmt.Sprintf("Cleanup done! removed %d plugin(s)", removed))
	}
	return nil
}
