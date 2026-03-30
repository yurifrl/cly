package splash

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "splash",
		Short: "Animated color gradient splash screen demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
