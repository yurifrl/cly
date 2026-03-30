package capability

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "capability",
		Short: "Query terminal capabilities demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
