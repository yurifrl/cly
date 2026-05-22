package zl

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
	"github.com/yurifrl/cly/pkg/envs"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

const pluginPath = "file:~/.config/zellij/plugins/zellij-switch.wasm"

// SwitchOpts holds parsed switch subcommand options.
type SwitchOpts struct {
	Session      string
	Cwd          string
	Layout       string
	Window       bool
	Interactive  bool
	SaveMapping  bool
}

// IsInsideZellij checks if running inside a Zellij session.
func IsInsideZellij() bool {
	return envs.InZellij()
}

// BuildPluginArgs builds the argument string for the zellij-switch plugin.
func BuildPluginArgs(opts SwitchOpts) string {
	parts := []string{"--session", opts.Session}
	if opts.Cwd != "" {
		parts = append(parts, "--cwd", opts.Cwd)
	}
	if opts.Layout != "" {
		parts = append(parts, "--layout", opts.Layout)
	}
	return strings.Join(parts, " ")
}

// BuildAttachArgs builds args for `zellij attach -c <name>`.
func BuildAttachArgs(opts SwitchOpts) []string {
	return []string{"attach", "-c", opts.Session}
}

// ParseSwitchFlags parses the switch subcommand arguments.
// Session name is a positional arg, flags are -c/--cwd, -l/--layout, -w/--window, -i/-z/--interactive, --save-mapping.
func ParseSwitchFlags(args []string) (SwitchOpts, error) {
	var opts SwitchOpts
	fs := pflag.NewFlagSet("switch", pflag.ContinueOnError)
	fs.StringVarP(&opts.Cwd, "cwd", "c", "", "Working directory")
	fs.StringVarP(&opts.Layout, "layout", "l", "", "Layout name")
	fs.BoolVarP(&opts.Window, "window", "w", false, "Open in new Ghostty window")
	fs.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive directory picker")
	fs.BoolVarP(&opts.Interactive, "zoxide", "z", false, "Interactive directory picker (alias)")
	fs.BoolVar(&opts.SaveMapping, "save-mapping", false, "Save session→dir mapping")

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	if remaining := fs.Args(); len(remaining) > 0 {
		opts.Session = remaining[0]
	}
	return opts, nil
}

// ParseCurrentSession extracts the current session name from `zellij list-sessions` output.
func ParseCurrentSession(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "current") {
			continue
		}
		clean := ansiRegex.ReplaceAllString(line, "")
		fields := strings.Fields(clean)
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

// ListSessions extracts all session names from `zellij list-sessions` output.
func ListSessions(output string) []string {
	var sessions []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		clean := ansiRegex.ReplaceAllString(line, "")
		fields := strings.Fields(clean)
		if len(fields) > 0 {
			sessions = append(sessions, fields[0])
		}
	}
	return sessions
}

// SessionExists checks if a session name exists in the list of sessions.
func SessionExists(sessionName string, sessions []string) bool {
	for _, s := range sessions {
		if s == sessionName {
			return true
		}
	}
	return false
}

// RunZellijPipe sends a payload to the switch plugin via `zellij pipe`.
func RunZellijPipe(payload string) error {
	cmd := exec.Command("zellij", "pipe", "--plugin", pluginPath, "--", payload)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GenerateCompletionsString returns fish completions for the zl command.
func GenerateCompletionsString() string {
	return `# Fish completions for zl (cly zl wrapper)
# Wrap zellij completions for passthrough
complete -c zl -n 'not __fish_seen_subcommand_from switch nuke' -w zellij

# Our custom switch subcommand
complete -c zl -f -n '__fish_use_subcommand' -a switch -d "Smart session switch/create"

# Session name completions: sessions + zoxide paths (deduplicated)
complete -c zl -n '__fish_seen_subcommand_from switch' -f -a "(
  zellij list-sessions --no-formatting --short 2>/dev/null
  zoxide query --list 2>/dev/null | head -20 | xargs -n1 basename
)" -d "Session name"

complete -c zl -n '__fish_seen_subcommand_from switch' -s c -l cwd -r -d "Working directory"
complete -c zl -n '__fish_seen_subcommand_from switch' -s l -l layout -r -d "Layout name" -a "(ls ~/.config/zellij/layouts/*.kdl 2>/dev/null | xargs -n1 basename | sed 's/.kdl\$//')"
complete -c zl -n '__fish_seen_subcommand_from switch' -s w -l window -d "Open in new Ghostty window"
complete -c zl -n '__fish_seen_subcommand_from switch' -s z -l zoxide -d "Interactive zoxide picker"
complete -c zl -n '__fish_seen_subcommand_from switch' -s i -l interactive -d "Interactive zoxide picker"
complete -c zl -n '__fish_seen_subcommand_from switch' -l save-mapping -d "Save session→dir mapping"

# Nuke current session
complete -c zl -f -n '__fish_use_subcommand' -a nuke -d "Kill current Zellij session"
`
}
