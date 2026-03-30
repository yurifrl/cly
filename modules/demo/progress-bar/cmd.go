package progressbar

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "progress-bar",
		Short: "Terminal progress bar (indeterminate) demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
