package lg_brightness

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-brightness",
		Short: "Lipgloss color brightness (lighten/darken) demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
