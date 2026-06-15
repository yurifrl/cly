// Package piwrap is a thin wrapper around the `pi` binary that adds a
// --name / -n flag. When --name is provided it propagates the session
// name through pkg/envs (which writes both the canonical and legacy
// env vars) and renames the current cmux tab to match. All other
// flags pass through to pi.
package piwrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/envs"
	"github.com/yurifrl/cly/pkg/helpy"
)

// defaultSessionFileNamePrefix is used when
// modules.piwrap.session_file_name_prefix is not set in config.
const defaultSessionFileNamePrefix = "cly"

// Run extracts piwrap-owned flags (--name/-n, --sety, --sety-string,
// --dry-run, --helpy) from args, applies their side effects, then
// execs `pi` with the remaining args. Returns the pi process exit
// error (if any). args should NOT include argv[0].
func Run(args []string) error {
	name, afterName := extractName(args)

	flags, setyErr := extractPiwrapFlags(afterName)
	if setyErr != nil {
		setyErr.Render()
		return setyErr
	}

	// --helpy short-circuits everything else.
	if flags.Helpy {
		renderHelpy(flags.HelpyJSON)
		return nil
	}

	rest := flags.Rest

	// Session import requires -n.
	if flags.Sety.HasImportID {
		if name == "" {
			e := newSetyError(CodeSetyNameRequired,
				"--sety session_import.id requires -n / --name", nil)
			e.Hint = "add -n <session-name>"
			e.Render()
			return e
		}
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("piwrap: getwd: %w", err)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("piwrap: home dir: %w", err)
		}
		plan, ierr := runImport(flags.Sety, name, home, cwd, flags.DryRun)
		if flags.DryRun {
			renderDryRun(plan, ierr)
			if ierr != nil {
				return ierr
			}
			if plan.BlockedBy != "" {
				// Plan completed but action would be blocked. Return
				// a non-error sentinel so cobra exits non-zero without
				// re-rendering anything.
				return newSetyError(plan.BlockedBy,
					"dry-run: action blocked by "+plan.BlockedBy, nil)
			}
			return nil
		}
		if ierr != nil {
			ierr.Render()
			return ierr
		}
		// Inject --session <target> so pi opens the freshly forked
		// file. hasSessionFlag still wins so explicit user --session
		// trumps the import target.
		if !hasSessionFlag(rest) {
			rest = append([]string{"--session", plan.Target}, rest...)
		}
	}

	if name != "" {
		if err := envs.SetSessionName(name); err != nil {
			return fmt.Errorf("piwrap: set session name: %w", err)
		}
		renameCmuxTab(name)

		if !hasSessionFlag(rest) {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("piwrap: getwd: %w", err)
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("piwrap: home dir: %w", err)
			}
			sessionPath := buildSessionPath(home, cwd, sessionFileNamePrefix(), name)
			if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
				return fmt.Errorf("piwrap: mkdir session dir: %w", err)
			}
			rest = append([]string{"--session", sessionPath}, rest...)
		}
	}

	piPath, err := exec.LookPath("pi")
	if err != nil {
		return errors.New("pi not found in PATH")
	}

	cmd := exec.Command(piPath, rest...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// pi inherits the ambient environment unchanged. Bedrock API-key
	// injection now lives in the pi `aws` extension (see DotFiles
	// pi-extensions), so it applies however pi is launched — not only
	// through this wrapper.

	return cmd.Run()
}

// renderHelpy writes the cheat sheet to stdout in text or JSON form.
func renderHelpy(asJSON bool) {
	if asJSON {
		_ = helpy.RenderJSON(os.Stdout, "", map[string]string{
			"pi_help":     "pi --help",
			"cobra_help":  "cly --help",
			"piwrap_help": "cly pi --help",
		})
		return
	}
	helpy.RenderText(os.Stdout,
		"cly pi \u2014 piwrap flags on top of the pi binary",
		"See also:\n  pi --help                     Passthrough flags (model, thinking, ...)\n  cly --help                    Cobra command tree.\n  cly pi --help                 This subcommand's stock help.\n",
	)
}

// renderDryRun emits the import plan as JSON on stdout. Includes the
// blocking error code (if any) under blocked_by.
func renderDryRun(plan importPlan, ierr *SetyError) {
	if ierr != nil && plan.BlockedBy == "" {
		plan.BlockedBy = ierr.Code
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]interface{}{
		"action": "session.import",
		"plan":   plan,
	})
}

// extractName scans args for --name/-n and returns (name, remaining).
// Supports: --name foo, --name=foo, -n foo, -n=foo. Removes the flag
// and its value from the returned slice. Only the first occurrence
// is consumed; subsequent ones pass through.
func extractName(args []string) (string, []string) {
	rest := make([]string, 0, len(args))
	name := ""
	consumed := false

	for i := 0; i < len(args); i++ {
		a := args[i]
		if consumed {
			rest = append(rest, a)
			continue
		}
		switch {
		case a == "--name" || a == "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
				consumed = true
			}
		case len(a) > 7 && a[:7] == "--name=":
			name = a[7:]
			consumed = true
		case len(a) > 3 && a[:3] == "-n=":
			name = a[3:]
			consumed = true
		default:
			rest = append(rest, a)
		}
	}
	return name, rest
}

// kebabCase converts an arbitrary name to lowercase kebab-case.
// Runs of non-alphanumeric characters collapse to a single dash;
// leading/trailing dashes are trimmed.
func kebabCase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevDash := true
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// encodeCwd mirrors pi's session directory naming: strip the leading
// path separator, replace remaining separators with '-', and wrap
// with '--' on both sides.
func encodeCwd(cwd string) string {
	trimmed := strings.TrimPrefix(cwd, string(filepath.Separator))
	replaced := strings.ReplaceAll(trimmed, string(filepath.Separator), "-")
	return "--" + replaced + "--"
}

// buildSessionPath returns the full path for a named cly session
// inside pi's session directory layout. The filename is
// "<prefix>-<kebab(name)>.jsonl".
func buildSessionPath(home, cwd, prefix, name string) string {
	return filepath.Join(
		home, ".pi", "agent", "sessions",
		encodeCwd(cwd),
		prefix+"-"+kebabCase(name)+".jsonl",
	)
}

// sessionFileNamePrefix returns the configured session file name
// prefix from modules.piwrap.session_file_name_prefix, falling back
// to the default when unset or blank.
func sessionFileNamePrefix() string {
	p := strings.TrimSpace(config.GetString("modules.piwrap.session_file_name_prefix"))
	if p == "" {
		return defaultSessionFileNamePrefix
	}
	return p
}

// hasSessionFlag reports whether args already contains --session in
// any supported form, so we don't override the user's choice.
func hasSessionFlag(args []string) bool {
	for _, a := range args {
		if a == "--session" || strings.HasPrefix(a, "--session=") {
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
	args := []string{"rename-tab"}
	if sid := envs.CmuxSurfaceID().Or(""); sid != "" {
		args = append(args, "--surface", sid)
	}
	args = append(args, title)
	cmd := exec.Command(cmuxPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cmux rename-tab failed: %v: %s\n", err, out)
	}
}
