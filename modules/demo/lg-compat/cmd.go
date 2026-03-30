package lg_compat

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-compat",
		Short: "Lipgloss v2 compat adaptive color demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
