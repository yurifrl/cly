package update

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// prepareBuild ensures everything needed by `go build` is fresh:
//   1. .env is up-to-date with .env.op (runs op inject when stale)
//   2. The embedded macOS notifier bundle is built when stale or missing
//
// Both checks are stale-aware. Subsequent runs are no-ops when nothing changed.
// Returns the env slice to feed into the build subprocess.
func prepareBuild(sourceDir string) ([]string, error) {
	if err := ensureEnv(sourceDir); err != nil {
		// Non-fatal: log and continue with whatever env we have. The build
		// can still work; just no codesign for the notifier on a fresh box.
		fmt.Fprintf(os.Stderr, "⚠️  env: %v\n", err)
	}
	if err := ensureNotifier(sourceDir); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  notifier: %v\n", err)
	}
	return loadEnvFile(sourceDir)
}

// ensureEnv runs `op inject` if .env is missing or older than .env.op.
func ensureEnv(sourceDir string) error {
	envPath := filepath.Join(sourceDir, ".env")
	tplPath := filepath.Join(sourceDir, ".env.op")

	tplStat, err := os.Stat(tplPath)
	if os.IsNotExist(err) {
		return nil // no template, nothing to do
	}
	if err != nil {
		return err
	}
	envStat, eerr := os.Stat(envPath)
	if eerr == nil && envStat.ModTime().After(tplStat.ModTime()) {
		return nil // .env is fresher than .env.op
	}
	if _, err := exec.LookPath("op"); err != nil {
		return fmt.Errorf("op CLI not on PATH; skipping .env refresh")
	}
	fmt.Println("⚡ refreshing .env from .env.op...")
	cmd := exec.Command("op", "inject",
		"--account", "my.1password.com",
		"-i", tplPath,
		"-o", envPath,
		"-f",
	)
	cmd.Dir = sourceDir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ensureNotifier runs pkg/notify/swift/build.sh when the embedded bundle is
// the committed placeholder OR any Swift source is newer than the tarball.
func ensureNotifier(sourceDir string) error {
	tarball := filepath.Join(sourceDir, "pkg", "notify", "assets", "cly-notifier.app.tar.gz")
	swiftDir := filepath.Join(sourceDir, "pkg", "notify", "swift")
	build := filepath.Join(swiftDir, "build.sh")

	if _, err := os.Stat(build); err != nil {
		return nil // notifier subsystem not present
	}

	st, err := os.Stat(tarball)
	stale := false
	switch {
	case err != nil:
		stale = true
	case st.Size() < 1024:
		stale = true // placeholder
	default:
		newest, _ := newestMtime(swiftDir)
		if newest.After(st.ModTime()) {
			stale = true
		}
	}
	if !stale {
		return nil
	}
	fmt.Println("⚡ rebuilding cly-notifier.app bundle...")
	cmd := exec.Command("bash", build)
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Ensure CLY_NOTIFIER_SIGN_ID from .env reaches build.sh.
	if extra, err := loadEnvFile(sourceDir); err == nil {
		cmd.Env = extra
	}
	return cmd.Run()
}

// newestMtime walks dir recursively and returns the latest mtime among
// regular files (excluding the .build/ artifacts directory).
func newestMtime(dir string) (time.Time, error) {
	var newest time.Time
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".build" {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest, err
}

// loadEnvFile parses sourceDir/.env and returns os.Environ() + extras.
func loadEnvFile(sourceDir string) ([]string, error) {
	env := os.Environ()
	envPath := filepath.Join(sourceDir, ".env")
	if _, err := os.Stat(envPath); err != nil {
		return env, nil
	}
	extras, err := parseEnvFile(envPath)
	if err != nil {
		return env, err
	}
	return append(env, extras...), nil
}

// parseEnvFile reads a simple KEY=VALUE file. Surrounding double-quotes on
// the value are stripped. Blank lines and lines starting with '#' are ignored.
func parseEnvFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "\"'")
		out = append(out, k+"="+v)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
