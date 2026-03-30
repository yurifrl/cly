package colorprofile

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "colorprofile",
		Short: "Color profile detection and true color demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
