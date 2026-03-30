package vanish

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "vanish",
		Short: "Program that vanishes without a trace on quit",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
