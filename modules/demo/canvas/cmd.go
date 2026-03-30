package canvas

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "canvas",
		Short: "Canvas card swapping demo with lipgloss compositor",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
