package obsidian

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:                "obsidian",
	Short:              "Obsidian CLI passthrough",
	DisableFlagParsing: true,
	RunE:               run,
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && args[0] == "capture" {
		return runCapture(args[1:])
	}
	return execObsidian(args)
}

func execObsidian(args []string) error {
	p, err := exec.LookPath("obsidian")
	if err != nil {
		return fmt.Errorf("obsidian not found in PATH")
	}
	return syscall.Exec(p, append([]string{"obsidian"}, args...), os.Environ())
}

func Register(parent *cobra.Command) {
	parent.AddCommand(Cmd)
}
