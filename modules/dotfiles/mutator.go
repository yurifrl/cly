package dotfiles

// Central mutation layer. Every filesystem write, deletion, and subprocess
// execution that changes observable state flows through this file. The
// --dry-run flag (wired in cmd.go) flips the package-level `dryRun` bool;
// wrappers below either perform the real action or log it and return nil.
//
// Read-only syscalls (Stat, ReadFile, ReadDir, Lstat, Readlink, Glob) do NOT
// belong here — they don't mutate anything and stay as direct os.* calls.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yurifrl/cly/pkg/style"
)

var dryRun bool

// SetDryRun toggles the package-level dry-run state. Only cmd.go should call this.
func SetDryRun(v bool) { dryRun = v }

// DryRun reports whether dry-run mode is active (for sites that need to
// short-circuit side-effects they cannot easily route through a wrapper).
func DryRun() bool { return dryRun }

func logDry(action, arg string) {
	fmt.Printf("%s %s %s\n", style.YellowStyle.Render("[dry-run]"), action, arg)
}

// ---- filesystem mutations ------------------------------------------------

func mutSymlink(oldname, newname string) error {
	if dryRun {
		logDry("symlink", fmt.Sprintf("%s -> %s", oldname, newname))
		return nil
	}
	return os.Symlink(oldname, newname)
}

func mutRemove(path string) error {
	if dryRun {
		logDry("rm", path)
		return nil
	}
	return os.Remove(path)
}

func mutRemoveAll(path string) error {
	if dryRun {
		logDry("rm -rf", path)
		return nil
	}
	return os.RemoveAll(path)
}

func mutMkdirAll(path string, mode os.FileMode) error {
	if dryRun {
		logDry("mkdir -p", path)
		return nil
	}
	return os.MkdirAll(path, mode)
}

func mutWriteFile(path string, data []byte, mode os.FileMode) error {
	if dryRun {
		logDry("write", fmt.Sprintf("%s (%d bytes)", path, len(data)))
		return nil
	}
	return os.WriteFile(path, data, mode)
}

func mutChmod(path string, mode os.FileMode) error {
	if dryRun {
		logDry("chmod", fmt.Sprintf("%s %o", path, mode))
		return nil
	}
	return os.Chmod(path, mode)
}

// ---- subprocess mutations ------------------------------------------------

// mutExec runs `name args...` and streams stdout/stderr. In dry-run the
// command is only logged.
func mutExec(name string, args ...string) error {
	if dryRun {
		logDry("exec", fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// mutExecDir is mutExec with a working directory.
func mutExecDir(dir, name string, args ...string) error {
	if dryRun {
		logDry("exec", fmt.Sprintf("(cd %s && %s %s)", dir, name, strings.Join(args, " ")))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
