// Package py consolidates all pi (coding agent) tools under one command.
package py

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/dotfiles"
	clypi "github.com/yurifrl/cly/modules/pi"
	pianon "github.com/yurifrl/cly/modules/pi-anon"
	pitree "github.com/yurifrl/cly/modules/pi-tree"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "py",
		Short: "Pi coding agent tools",
	}

	pitree.Register(cmd)
	pianon.Register(cmd)
	clypi.Register(cmd)
	cmd.AddCommand(modelsCmd(), secCmd(), settingsCmd(), updateCmd())

	parent.AddCommand(cmd)
}

func modelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "Pick a pi model with gum and launch pi",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := exec.Command("pi", "--list-models").Output()
			if err != nil {
				return fmt.Errorf("pi --list-models: %w", err)
			}
			var models []string
			for i, line := range strings.Split(string(out), "\n") {
				if i == 0 || line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					models = append(models, fields[0]+"/"+fields[1])
				}
			}
			if len(models) == 0 {
				return fmt.Errorf("no models found — is pi installed?")
			}

			filter := exec.Command("gum", "filter",
				"--header=  Pick a model — Enter to launch · Tab to copy",
				"--placeholder=Search models...",
				"--height=20", "--width=70",
				"--prompt=  ", "--indicator=▶")
			filter.Stdin = strings.NewReader(strings.Join(models, "\n"))
			filter.Stderr = os.Stderr
			sel, err := filter.Output()
			if err != nil || len(strings.TrimSpace(string(sel))) == 0 {
				return nil
			}
			model := strings.TrimSpace(string(sel))

			choose := exec.Command("gum", "choose",
				"--header=  Model: "+model,
				"--height=4",
				"Launch pi --model "+model,
				"Copy to clipboard")
			choose.Stderr = os.Stderr
			action, err := choose.Output()
			if err != nil || len(strings.TrimSpace(string(action))) == 0 {
				return nil
			}

			if strings.HasPrefix(strings.TrimSpace(string(action)), "Launch") {
				c := exec.Command("pi", append([]string{"--model", model}, args...)...)
				c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
				return c.Run()
			}
			pb := exec.Command("pbcopy")
			pb.Stdin = strings.NewReader(model)
			return pb.Run()
		},
	}
}

func secCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sec",
		Short: "Pick an AIHub model and launch pi",
		RunE: func(cmd *cobra.Command, args []string) error {
			choose := exec.Command("gum", "choose",
				"--header=Select AIHub model:",
				"aihub/claude-opus-4-6",
				"aihub/claude-sonnet-4-5-20250929",
				"aihub/gemini-2.5-flash")
			choose.Stderr = os.Stderr
			out, err := choose.Output()
			if err != nil || len(strings.TrimSpace(string(out))) == 0 {
				return nil
			}
			model := strings.TrimSpace(string(out))
			c := exec.Command("pi", append([]string{"--model", model}, args...)...)
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
			return c.Run()
		},
	}
}

func settingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "settings",
		Short: "Edit pi agent settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			src := home + "/DotFiles/home/.pi/agent/settings.jsonc"
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}
			c := exec.Command(editor, src)
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := c.Run(); err != nil {
				return err
			}
			m, err := dotfiles.ParseMapping("./home/.pi/agent/settings.jsonc -> ~/.pi/agent/settings.json")
			if err != nil {
				return err
			}
			dotfiles.CopyJsoncToJson(m)
			return nil
		},
	}
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update pi and global pnpm packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, c := range []*exec.Cmd{
				pipe(exec.Command("pnpm", "update", "-g", "-L")),
				pipe(exec.Command("pi", "update")),
			} {
				if err := c.Run(); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func pipe(c *exec.Cmd) *exec.Cmd {
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c
}
