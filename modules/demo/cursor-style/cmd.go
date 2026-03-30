package cursorstyle

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "cursor-style",
		Short: "Cursor style switching demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
