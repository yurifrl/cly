package setterminalcolor

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "set-terminal-color",
		Short: "Set terminal foreground, background, and cursor colors",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
