package bundle

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// A `release:` Ternfile entry downloads a GitHub release tarball that ships a
// plugin together with its binaries (e.g. plugin/ + bin/), unpacks it under
// ternReleaseRoot()/<id>/<tag>/ and links <subpath> from there. Because the
// whole archive is kept, a plugin that expects ../bin/ next to itself works.
//
//	<id>  release:github.com/OWNER/REPO/SUBPATH[@<tag|latest>]
//
// The archive must be a .tar.gz named *_<goos>_<goarch>.tar.gz with a single
// top-level directory, which is stripped.

// ternReleaseRoot is where release archives are unpacked. Tests override it.
var ternReleaseRoot = func() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "cly", "tern")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cly", "tern")
}

// setReleasePaths fills root and dir once the tag is known.
func (e *ternEntry) setReleasePaths() {
	if !e.release || e.tag == "" {
		return
	}
	e.root = filepath.Join(ternReleaseRoot(), e.id, e.tag)
	e.dir = filepath.Join(e.root, filepath.FromSlash(e.subpath))
}

// ternAsset finds the download URL of repo@tag's tarball for this platform.
func ternAsset(repo, tag string) (string, error) {
	req, err := http.NewRequest("GET", ternGitHubAPI+"/repos/"+repo+"/releases/tags/"+tag, nil)
	if err != nil {
		return "", err
	}
	setGitHubAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release %s@%s: HTTP %d", repo, tag, resp.StatusCode)
	}
	var rel struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	suffix := fmt.Sprintf("_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, suffix) {
			return a.URL, nil
		}
	}
	return "", fmt.Errorf("release %s@%s has no asset matching *%s", repo, tag, suffix)
}

func setGitHubAuth(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if tok := os.Getenv(k); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
			return
		}
	}
}

// fetchTernRelease unpacks the release into e.root, replacing any previous
// copy, and drops other tags' directories for the same plugin.
func fetchTernRelease(e ternEntry) error {
	url, err := ternAsset(e.repo, e.tag)
	if err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	parent := filepath.Dir(e.root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".fetch-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := untarStrip1(resp.Body, tmp); err != nil {
		return err
	}
	if !isDir(filepath.Join(tmp, filepath.FromSlash(e.subpath))) {
		return fmt.Errorf("%s@%s has no directory %q", e.repo, e.tag, e.subpath)
	}
	if err := os.RemoveAll(e.root); err != nil {
		return err
	}
	if err := os.Rename(tmp, e.root); err != nil {
		return err
	}
	old, _ := os.ReadDir(parent)
	for _, o := range old {
		if o.IsDir() && o.Name() != e.tag {
			os.RemoveAll(filepath.Join(parent, o.Name()))
		}
	}
	return nil
}

// untarStrip1 extracts a .tar.gz into dst, dropping the first path component
// and refusing entries that would escape dst.
func untarStrip1(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		_, rel, ok := strings.Cut(filepath.ToSlash(h.Name), "/")
		if !ok || rel == "" {
			continue
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if target != dst && !strings.HasPrefix(target, dst+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %s", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, h.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		}
	}
}
