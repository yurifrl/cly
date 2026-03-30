package printkey

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "print-key",
		Short: "Print key press details demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
