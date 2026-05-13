// Package mut is a thin, centralized mutation layer. It wraps every
// filesystem write / deletion / subprocess execution behind functions that
// either perform the action or — when dry-run mode is enabled — log the
// intended action and return nil.
//
// Use this package whenever you want a single switch (the --dry-run flag)
// to turn a program into a preview of what it would do, without having to
// thread a "dryRun" parameter through every call site.
//
// Typical wiring:
//
//	// cmd.go
//	var dryRun bool
//	cmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "preview only")
//	cmd.PersistentPreRun = func(_ *cobra.Command, _ []string) {
//	    mut.Init(mut.Options{DryRun: dryRun})
//	}
//
//	// anywhere in your code
//	if err := mut.MkdirAll(dir, 0755); err != nil { ... }
//	if err := mut.Exec("op", "inject", "-i", src, "-o", dst); err != nil { ... }
//
// Read-only syscalls (Stat, ReadFile, ReadDir, Lstat, Readlink, Glob, ...)
// intentionally do NOT live here — they don't mutate anything and should be
// called directly from the os package.
package mut

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Options configures the mutation layer. All fields are optional.
type Options struct {
	// DryRun, when true, causes every mutating call to log the intended action
	// (via Logger) and return nil without touching the filesystem or spawning
	// any subprocess.
	DryRun bool

	// Logger receives dry-run action strings. Defaults to a Stdout logger that
	// prints "[dry-run] <action> <target>" lines.
	Logger func(action, target string)

	// Stdout/Stderr are used as the streams for Exec/ExecDir when not dry-run.
	// Defaults to os.Stdout / os.Stderr.
	Stdout io.Writer
	Stderr io.Writer
}

var (
	mu     sync.RWMutex
	opts   Options
	active bool
)

// Init configures the package-level state. Safe to call multiple times; the
// most recent call wins. If Logger/Stdout/Stderr are nil, defaults are used.
func Init(o Options) {
	if o.Logger == nil {
		o.Logger = defaultLogger
	}
	if o.Stdout == nil {
		o.Stdout = os.Stdout
	}
	if o.Stderr == nil {
		o.Stderr = os.Stderr
	}
	mu.Lock()
	opts = o
	active = true
	mu.Unlock()
}

// SetDryRun is a shortcut for toggling only the DryRun flag. If Init has not
// been called yet, defaults are applied.
func SetDryRun(v bool) {
	mu.Lock()
	if !active {
		mu.Unlock()
		Init(Options{DryRun: v})
		return
	}
	opts.DryRun = v
	mu.Unlock()
}

// DryRun reports whether dry-run mode is currently enabled.
func DryRun() bool {
	mu.RLock()
	defer mu.RUnlock()
	return opts.DryRun
}

func snapshot() Options {
	mu.RLock()
	defer mu.RUnlock()
	if !active {
		return Options{Logger: defaultLogger, Stdout: os.Stdout, Stderr: os.Stderr}
	}
	return opts
}

func defaultLogger(action, target string) {
	fmt.Printf("[dry-run] %s %s\n", action, target)
}

func log(action, target string) {
	snapshot().Logger(action, target)
}

// ---- filesystem mutations ------------------------------------------------

// Symlink creates a new symlink `newname` pointing at `oldname`.
func Symlink(oldname, newname string) error {
	if DryRun() {
		log("symlink", fmt.Sprintf("%s -> %s", oldname, newname))
		return nil
	}
	return os.Symlink(oldname, newname)
}

// Remove deletes a single file or empty directory.
func Remove(path string) error {
	if DryRun() {
		log("rm", path)
		return nil
	}
	return os.Remove(path)
}

// RemoveAll deletes a path and any children it contains.
func RemoveAll(path string) error {
	if DryRun() {
		log("rm -rf", path)
		return nil
	}
	return os.RemoveAll(path)
}

// MkdirAll creates a directory and any missing parents.
func MkdirAll(path string, mode os.FileMode) error {
	if DryRun() {
		log("mkdir -p", path)
		return nil
	}
	return os.MkdirAll(path, mode)
}

// WriteFile writes `data` to `path` with the given permission mode.
func WriteFile(path string, data []byte, mode os.FileMode) error {
	if DryRun() {
		log("write", fmt.Sprintf("%s (%d bytes)", path, len(data)))
		return nil
	}
	return os.WriteFile(path, data, mode)
}

// Chmod changes the mode of `path`.
func Chmod(path string, mode os.FileMode) error {
	if DryRun() {
		log("chmod", fmt.Sprintf("%s %o", path, mode))
		return nil
	}
	return os.Chmod(path, mode)
}

// Rename moves/renames a file or directory.
func Rename(oldpath, newpath string) error {
	if DryRun() {
		log("mv", fmt.Sprintf("%s -> %s", oldpath, newpath))
		return nil
	}
	return os.Rename(oldpath, newpath)
}

// ---- subprocess mutations ------------------------------------------------

// Exec runs `name args...` streaming stdout/stderr. In dry-run it only logs.
func Exec(name string, args ...string) error {
	if DryRun() {
		log("exec", fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
		return nil
	}
	o := snapshot()
	cmd := exec.Command(name, args...)
	cmd.Stdout = o.Stdout
	cmd.Stderr = o.Stderr
	return cmd.Run()
}

// ExecDir is Exec with a working directory.
func ExecDir(dir, name string, args ...string) error {
	if DryRun() {
		log("exec", fmt.Sprintf("(cd %s && %s %s)", dir, name, strings.Join(args, " ")))
		return nil
	}
	o := snapshot()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = o.Stdout
	cmd.Stderr = o.Stderr
	return cmd.Run()
}

// Log records a custom action in the dry-run log. Use for side-effects that
// aren't covered by the built-in wrappers (e.g. network downloads). When
// dry-run is off this is a no-op — caller must still perform the action.
// Returns DryRun() so callers can use it as a guard:
//
//	if mut.Log("download", url) { return nil }
//	// ... actual download ...
func Log(action, target string) bool {
	if !DryRun() {
		return false
	}
	log(action, target)
	return true
}
