package bundle

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Ternfile.lock records what the last sync installed: one line per plugin
// with its id, its Ternfile source and the tag or commit in place. It is a
// record, not a pin: github: entries keep following upstream, and the lock
// is rewritten after each sync so version moves show up in git diffs.
//
// It also caches probed ids. An entry written without an id gets it from
// the plugin's plugin.toml at the resolved ref; a tag or commit never
// changes, so (source, ref) -> id is looked up in the lock first and only
// fetched from GitHub for a new ref.

const ternLockHeader = "# Ternfile.lock — written by `cly bundle tern` after each sync. Do not edit.\n" +
	"# Records what is installed (id, source, tag/commit); it does not pin versions.\n"

// ternLockEntry is one Ternfile.lock line.
type ternLockEntry struct {
	id     string
	source string // Ternfile source as written (with +bin folded in as a suffix)
	ref    string // tag or commit; "-" when untracked (links, plain sources)
}

func ternLockPath(bundleFile string) string {
	return expandPath(bundleFile) + ".lock"
}

// lockSource is the key an entry is recorded under: its source as written,
// plus the +bin option since it changes what gets installed.
func (e ternEntry) lockSource() string {
	src := e.raw
	if src == "" {
		src = e.source
	}
	if e.bin {
		return src + " +bin"
	}
	return src
}

// readTernLock loads the lock; a missing file is an empty lock.
func readTernLock(path string) ([]ternLockEntry, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []ternLockEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch {
		case len(fields) == 3:
			out = append(out, ternLockEntry{id: fields[0], source: fields[1], ref: fields[2]})
		case len(fields) == 4 && fields[2] == "+bin":
			out = append(out, ternLockEntry{id: fields[0], source: fields[1] + " +bin", ref: fields[3]})
		default:
			return nil, fmt.Errorf("%s: malformed line %q", filepath.Base(path), line)
		}
	}
	return out, sc.Err()
}

// writeTernLock writes entries in order, aligned, and only touches the file
// when the content changes.
func writeTernLock(path string, entries []ternLockEntry) error {
	idW, srcW := 0, 0
	for _, e := range entries {
		idW, srcW = max(idW, len(e.id)), max(srcW, len(e.source))
	}
	var b strings.Builder
	b.WriteString(ternLockHeader + "\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%-*s  %-*s  %s\n", idW, e.id, srcW, e.source, e.ref)
	}
	if old, err := os.ReadFile(path); err == nil && string(old) == b.String() {
		return nil
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// pluginTOMLID returns the top-level id of a plugin.toml. Blocks and lenses
// carry their own `id =` lines inside [[sections]], so reading stops at the
// first section header.
func pluginTOMLID(r io.Reader) string {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			return ""
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "id" {
			continue
		}
		val = strings.TrimSpace(val)
		if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		return strings.Trim(val, `"'`)
	}
	return ""
}

// probeTernID fetches PATH/plugin.toml from repo at ref and returns its id.
func probeTernID(repo, subpath, ref string) (string, error) {
	file := strings.TrimPrefix(strings.Trim(subpath, "/")+"/plugin.toml", "/")
	req, err := http.NewRequest("GET", ternGitHubAPI+"/repos/"+repo+"/contents/"+file+"?ref="+ref, nil)
	if err != nil {
		return "", err
	}
	setGitHubAuth(req)
	req.Header.Set("Accept", "application/vnd.github.raw")
	resp, err := ternHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", repo, errTernOffline)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no %s in %s at %s; write the id in front of the source", file, repo, shortRef(ref))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("probe id: %s/%s: HTTP %d", repo, file, resp.StatusCode)
	}
	id := pluginTOMLID(resp.Body)
	if id == "" {
		return "", fmt.Errorf("%s in %s has no top-level id; write the id in front of the source", file, repo)
	}
	return id, nil
}

// resolveTernIDs fills the id of entries written without one: links read
// their local plugin.toml, github: entries use the lock's (source, ref)
// cache or probe GitHub. When that fails, the lock's last known id for the
// source keeps the entry matched to its installed plugin and the error is
// shown on the row. Runs after resolveTernRefs so refs are known.
func resolveTernIDs(entries []ternEntry, lock []ternLockEntry) error {
	cached := map[string]string{} // source\x00ref -> id
	lastID := map[string]string{} // source -> id
	for _, l := range lock {
		cached[l.source+"\x00"+l.ref] = l.id
		lastID[l.source] = l.id
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i := range entries {
		e := &entries[i]
		if e.id != "" {
			continue
		}
		switch {
		case e.link:
			if f, err := os.Open(filepath.Join(e.source, "plugin.toml")); err == nil {
				e.id = pluginTOMLID(f)
				f.Close()
			}
			if e.id == "" {
				e.id = lastID[e.lockSource()]
			}
			if e.id == "" && !e.optional {
				e.err = fmt.Errorf("no plugin.toml id in %s; write the id in front of the source", e.source)
			}
		case e.github && e.want != "" && e.err == nil:
			if id, ok := cached[e.lockSource()+"\x00"+e.want]; ok {
				e.id = id
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				id, err := probeTernID(e.repo, e.subpath, e.want)
				if err != nil {
					e.id, e.err = lastID[e.lockSource()], err
					return
				}
				e.id = id
			}()
		default:
			// Ref did not resolve (error already on the entry): fall back to
			// the last recorded id so the row still names its plugin.
			e.id = lastID[e.lockSource()]
		}
	}
	wg.Wait()

	seen := map[string]string{}
	for _, e := range entries {
		if e.id == "" {
			continue
		}
		if prev, dup := seen[e.id]; dup {
			return fmt.Errorf("Ternfile: %s and %s are both plugin %q", prev, e.lockSource(), e.id)
		}
		seen[e.id] = e.lockSource()
	}
	return nil
}

// ternLockAfterSync builds the lock from the synced state: installed copies
// report the ref in their marker, +bin entries their tag, links "-".
// Entries that failed keep their previous lock line, so ids stay cached.
func ternLockAfterSync(entries []ternEntry, installed map[string]ternInstalled, failed map[string]bool, prev []ternLockEntry) []ternLockEntry {
	prevBySource := map[string]ternLockEntry{}
	for _, l := range prev {
		prevBySource[l.source] = l
	}
	var out []ternLockEntry
	for _, e := range entries {
		have, ok := installed[e.id]
		if e.id == "" || failed[e.id] || !ok || e.err != nil {
			if l, found := prevBySource[e.lockSource()]; found {
				out = append(out, l)
			}
			continue
		}
		ref := "-"
		switch {
		case e.bin && e.tag != "":
			ref = e.tag
		case !have.link && have.ref != "":
			ref = have.ref
		}
		out = append(out, ternLockEntry{id: e.id, source: e.lockSource(), ref: ref})
	}
	return out
}
