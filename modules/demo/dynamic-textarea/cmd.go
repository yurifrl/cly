package dynamictextarea

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "dynamic-textarea",
		Short: "Dynamic height textarea demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
