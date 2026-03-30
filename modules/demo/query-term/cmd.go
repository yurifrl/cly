package queryterm

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "query-term",
		Short: "Query terminal with ANSI sequences demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
