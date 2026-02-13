package zl

import (
	"os"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

const pluginPath = "file:~/.config/zellij/plugins/zellij-switch.wasm"

// SwitchOpts holds parsed switch subcommand options.
type SwitchOpts struct {
	Session string
	Cwd     string
	Layout  string
	Window  bool
}

// IsInsideZellij checks if running inside a Zellij session.
func IsInsideZellij() bool {
	return os.Getenv("ZELLIJ") != ""
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
// Session name is a positional arg, flags are -c/--cwd, -l/--layout, -w/--window.
func ParseSwitchFlags(args []string) (SwitchOpts, error) {
	var opts SwitchOpts
	fs := pflag.NewFlagSet("switch", pflag.ContinueOnError)
	fs.StringVarP(&opts.Cwd, "cwd", "c", "", "Working directory")
	fs.StringVarP(&opts.Layout, "layout", "l", "", "Layout name")
	fs.BoolVarP(&opts.Window, "window", "w", false, "Open in new Ghostty window")

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

// GenerateCompletionsString returns fish completions for the zl command.
func GenerateCompletionsString() string {
	return `# Fish completions for zl (cly zl wrapper)
# Wrap zellij completions for passthrough
complete -c zl -n 'not __fish_seen_subcommand_from switch nuke' -w zellij

# Our custom switch subcommand
complete -c zl -f -n '__fish_use_subcommand' -a switch -d "Smart session switch/create"
complete -c zl -n '__fish_seen_subcommand_from switch' -f -a "(zellij list-sessions --no-formatting --short 2>/dev/null)" -d "Session name"
complete -c zl -n '__fish_seen_subcommand_from switch' -s c -l cwd -r -d "Working directory"
complete -c zl -n '__fish_seen_subcommand_from switch' -s l -l layout -r -d "Layout name" -a "(ls ~/.config/zellij/layouts/*.kdl 2>/dev/null | xargs -n1 basename | sed 's/.kdl\$//')"
complete -c zl -n '__fish_seen_subcommand_from switch' -s w -l window -d "Open in new Ghostty window"

# Nuke current session
complete -c zl -f -n '__fish_use_subcommand' -a nuke -d "Kill current Zellij session"
`
}
