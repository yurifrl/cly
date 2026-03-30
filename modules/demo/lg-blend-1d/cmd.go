package lg_blend_1d

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-blend-1d",
		Short: "Lipgloss 1D color gradient blending demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
