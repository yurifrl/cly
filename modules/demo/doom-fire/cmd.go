package doomfire

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "doom-fire",
		Short: "DOOM fire effect demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
