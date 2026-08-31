// Package ompwrap is a thin wrapper around the `omp` binary (pi v18) that
// adds a --name / -n flag. When --name is provided it propagates the
// session name through pkg/envs (which writes both the canonical and
// legacy env vars) and renames the current cmux tab to match. All other
// flags pass through to omp.
package ompwrap

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/envs"
)

// defaultSessionFileNamePrefix is used when
// modules.ompwrap.session_file_name_prefix is not set in config.
const defaultSessionFileNamePrefix = "cly"

// Run extracts ompwrap-owned flags (--name/-n) from args, applies their
// side effects, then execs `omp` with the remaining args. Returns the
// omp process exit error (if any). args should NOT include argv[0].
func Run(args []string) error {
	name, rest := extractName(args)

	if name != "" {
		if err := envs.SetSessionName(name); err != nil {
			return fmt.Errorf("ompwrap: set session name: %w", err)
		}
		renameCmuxTab(name)

		if !hasSessionFlag(rest) {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("ompwrap: getwd: %w", err)
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("ompwrap: home dir: %w", err)
			}
			sessionPath := buildSessionPath(home, cwd, sessionFileNamePrefix(), name)
			if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
				return fmt.Errorf("ompwrap: mkdir session dir: %w", err)
			}
			rest = append([]string{"--resume", sessionPath}, rest...)
		}
	}

	ompPath, err := exec.LookPath("omp")
	if err != nil {
		return errors.New("omp not found in PATH")
	}

	cmd := exec.Command(ompPath, rest...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// omp inherits the ambient environment unchanged.

	return cmd.Run()
}

// extractName scans args for --name/-n and returns (name, remaining).
// Supports: --name foo, --name=foo, -n foo, -n=foo. Removes the flag
// and its value from the returned slice. Only the first occurrence
// is consumed; subsequent ones pass through.
func extractName(args []string) (string, []string) {
	name := ""
	rest := make([]string, 0, len(args))
	consumed := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case !consumed && (arg == "--name" || arg == "-n") && i+1 < len(args):
			name = args[i+1]
			consumed = true
			i++
		case !consumed && strings.HasPrefix(arg, "--name="):
			name = strings.TrimPrefix(arg, "--name=")
			consumed = true
		case !consumed && strings.HasPrefix(arg, "-n="):
			name = strings.TrimPrefix(arg, "-n=")
			consumed = true
		default:
			rest = append(rest, arg)
		}
	}
	return name, rest
}

// kebabCase converts an arbitrary name to lowercase kebab-case.
// Runs of non-alphanumeric characters collapse to a single dash;
// leading/trailing dashes are trimmed.
func kebabCase(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// encodeCwd mirrors omp's session directory naming: strip the leading
// path separator, replace remaining separators with '-', and wrap
// with '-' on the left and '--' on the right (observed omp v18 layout:
// "-Workdir-Yuri-cly"). pi uses "--<encoded>--"; the differing suffix
// keeps the two providers' dirs distinct in their own roots.
func encodeCwd(cwd string) string {
	trimmed := strings.Trim(cwd, "/")
	encoded := strings.ReplaceAll(trimmed, "/", "-")
	return "-" + encoded
}

// buildSessionPath returns the full path for a named cly session
// inside omp's session directory layout. The filename is
// "<prefix>-<kebab(name)>.jsonl".
func buildSessionPath(home, cwd, prefix, name string) string {
	return filepath.Join(
		home, ".omp", "agent", "sessions",
		encodeCwd(cwd),
		prefix+"-"+kebabCase(name)+".jsonl",
	)
}

// sessionFileNamePrefix returns the configured session file name
// prefix from modules.ompwrap.session_file_name_prefix, falling back
// to the default when unset or blank.
func sessionFileNamePrefix() string {
	p := strings.TrimSpace(configGetString("modules.ompwrap.session_file_name_prefix"))
	if p == "" {
		return defaultSessionFileNamePrefix
	}
	return p
}

// hasSessionFlag reports whether args already contains --resume or
// --session in any supported form, so we don't override the user's
// choice.
func hasSessionFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--resume" || arg == "-r" ||
			strings.HasPrefix(arg, "--resume=") ||
			arg == "--session" ||
			strings.HasPrefix(arg, "--session=") {
			return true
		}
	}
	return false
}

// renameCmuxTab renames the current cmux tab. Best-effort: silently
// skips if cmux isn't on PATH or the call fails (e.g., not running
// inside cmux).
func renameCmuxTab(title string) {
	cmuxPath, err := exec.LookPath("cmux")
	if err != nil {
		return
	}
	_ = exec.Command(cmuxPath, "tab", "rename", title).Run()
}
