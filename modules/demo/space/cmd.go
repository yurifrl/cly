package space

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Space-like moving background FPS demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
