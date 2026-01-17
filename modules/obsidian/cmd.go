package obsidian

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "obsidian",
	Short: "Obsidian helpers via Claude",
	Long:  "Commands for interacting with Obsidian through Claude Code",
}

func Register(parent *cobra.Command) {
	parent.AddCommand(Cmd)
	registerCapture(Cmd)
}
