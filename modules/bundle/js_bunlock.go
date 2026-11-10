package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// bun pins every resolved tarball URL in its global lockfile and reuses
// those URLs on later installs, ignoring --registry for packages already in
// the lock. Entries resolved through a user default registry (e.g. a company
// proxy from ~/.npmrc) therefore keep hitting that registry. Rewriting their
// URL prefix to jsRegistry() moves them over; both registries share the
// `<base>/<pkg>/-/<file>.tgz` layout and serve identical tarballs, so the
// integrity hashes stay valid. Scoped registries (e.g. GitHub Packages) use
// a different base and are left untouched.

// jsUserDefaultRegistry returns the default registry the user's own config
// would route plain installs through, or "" if none is set.
func jsUserDefaultRegistry() string {
	for _, env := range []string{"BUN_CONFIG_REGISTRY", "NPM_CONFIG_REGISTRY", "npm_config_registry"} {
		if r := strings.TrimSpace(os.Getenv(env)); r != "" {
			return r
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, path := range []string{filepath.Join(home, ".bunfig.toml"), filepath.Join(home, ".npmrc")} {
			if r := registryFromNpmrcLike(path); r != "" {
				return r
			}
		}
	}
	return ""
}

// registryFromNpmrcLike scans a bunfig.toml/.npmrc file for a top-level
// `registry = value` entry (scoped `@scope:registry` lines are ignored).
func registryFromNpmrcLike(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// bunfig.toml writes `install.registry = value`; .npmrc writes
		// `registry=value`. Both reduce to the same key.
		line = strings.TrimPrefix(line, "install.")
		if !strings.HasPrefix(line, "registry") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "registry"))
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		rest = strings.Trim(strings.TrimSpace(rest[1:]), `"'`)
		if rest != "" {
			return rest
		}
	}
	return ""
}

// bunGlobalLockPath returns the path of bun's global-install lockfile.
func bunGlobalLockPath() string {
	root := os.Getenv("BUN_INSTALL")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".bun")
	}
	return filepath.Join(root, "install", "global", "bun.lock")
}

// rewriteLockRegistry replaces every quoted tarball URL starting with
// from/ by to/ in the lockfile at path. Returns the number of rewritten
// URLs; a missing lockfile is not an error.
func rewriteLockRegistry(path, from, to string) (int, error) {
	from = strings.TrimSuffix(strings.TrimSpace(from), "/")
	to = strings.TrimSuffix(strings.TrimSpace(to), "/")
	if from == "" || to == "" || from == to {
		return 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	old, repl := `"`+from+`/`, `"`+to+`/`
	n := strings.Count(string(data), old)
	if n == 0 {
		return 0, nil
	}
	out := strings.ReplaceAll(string(data), old, repl)
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, []byte(out), info.Mode().Perm()); err != nil {
		return 0, err
	}
	return n, nil
}

// migrateBunLock moves bun global-lock entries off the user's default
// registry onto jsRegistry(). No-op for other managers.
func (b *JsBundler) migrateBunLock() {
	if b.manager != "bun" {
		return
	}
	from, to := jsUserDefaultRegistry(), jsRegistry()
	n, err := rewriteLockRegistry(bunGlobalLockPath(), from, to)
	if err != nil {
		printYellow(fmt.Sprintf("Warning: could not migrate bun global lockfile to %s: %v", to, err))
		return
	}
	if n > 0 {
		printYellow(fmt.Sprintf("Repointed %d bun lockfile URLs from %s to %s", n, from, to))
	}
}
