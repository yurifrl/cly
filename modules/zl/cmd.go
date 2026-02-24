package zl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/completion"
)

func Register(parent *cobra.Command) {
	completion.RegisterAlias("zl", GenerateCompletionsString())

	cmd := &cobra.Command{
		Use:                "zl [args...]",
		Short:              "Zellij wrapper with smart session switching",
		Long:               "Drop-in replacement for zellij. All args pass through except 'switch'.",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE:               run,
	}
	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "switch":
			return runSwitch(args[1:])
		case "nuke":
			return runNuke()
		}
	}
	return execZellij(args)
}

func runSwitch(args []string) error {
	opts, err := ParseSwitchFlags(args)
	if err != nil {
		return err
	}

	// Handle interactive mode
	if opts.Interactive {
		dir, err := QueryZoxideInteractive()
		if err != nil {
			return fmt.Errorf("interactive directory selection failed: %w", err)
		}
		if dir == "" {
			return fmt.Errorf("no directory selected")
		}
		opts.Cwd = dir

		// Infer session name from directory basename if not provided
		if opts.Session == "" {
			opts.Session = getBasename(dir)
		}
	}

	if opts.Session == "" {
		return fmt.Errorf("session name required: cly zl switch <session-name> [-c cwd] [-l layout] [-w] [-i|-z] [--save-mapping]")
	}

	if IsInsideZellij() {
		return switchInside(opts)
	}
	return switchOutside(opts)
}

func switchInside(opts SwitchOpts) error {
	return openGhosttyWindow(opts)
}

func switchOutside(opts SwitchOpts) error {
	// Resolve directory if not explicitly provided
	if opts.Cwd == "" {
		opts.Cwd = ResolveDirectory(opts.Session)
	}

	if opts.Window {
		return openGhosttyWindow(opts)
	}

	attachArgs := BuildAttachArgs(opts)
	if opts.Cwd != "" {
		if err := os.Chdir(opts.Cwd); err != nil {
			return fmt.Errorf("failed to change to directory %s: %w", opts.Cwd, err)
		}

		// Update zoxide database if enabled
		cfg := LoadZlConfig()
		if cfg.UpdateZoxide {
			_ = UpdateZoxide(opts.Cwd)
		}

		// Save mapping if requested
		if opts.SaveMapping {
			if err := SaveSessionMapping(opts.Session, opts.Cwd); err != nil {
				return fmt.Errorf("failed to save mapping: %w", err)
			}
		}
	}

	return execZellij(attachArgs)
}

func openGhosttyWindow(opts SwitchOpts) error {
	attachArgs := BuildAttachArgs(opts)
	zellijArgs := append([]string{"zellij"}, attachArgs...)

	ghosttyPath, err := exec.LookPath("ghostty")
	if err != nil {
		return fmt.Errorf("ghostty not found in PATH")
	}

	allArgs := append([]string{"-e"}, zellijArgs...)
	c := exec.Command(ghosttyPath, allArgs...)
	if opts.Cwd != "" {
		c.Dir = opts.Cwd
	}
	return c.Start()
}

func runNuke() error {
	if !IsInsideZellij() {
		return fmt.Errorf("not inside a Zellij session")
	}

	out, err := exec.Command("zellij", "list-sessions").Output()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	session := ParseCurrentSession(string(out))
	if session == "" {
		return fmt.Errorf("could not determine current session")
	}

	return execZellij([]string{"kill-session", session})
}

func execZellij(args []string) error {
	zellijPath, err := exec.LookPath("zellij")
	if err != nil {
		return fmt.Errorf("zellij not found in PATH")
	}
	return syscall.Exec(zellijPath, append([]string{"zellij"}, args...), os.Environ())
}

func getBasename(path string) string {
	return filepath.Base(path)
}
