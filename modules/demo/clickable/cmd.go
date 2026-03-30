package clickable

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "clickable",
		Short: "Clickable and draggable dialog boxes demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
