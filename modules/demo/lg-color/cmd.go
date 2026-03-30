package lg_color

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-color",
		Short: "Lipgloss adaptive color detection demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
