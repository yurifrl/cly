package lg_blend_rotation

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-blend-rotation",
		Short: "Lipgloss animated border blend rotation demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
