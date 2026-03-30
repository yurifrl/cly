package lg_blend_2d

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-blend-2d",
		Short: "Lipgloss 2D color gradient blending demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
