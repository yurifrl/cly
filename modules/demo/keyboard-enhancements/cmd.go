package keyboardenhancements

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "keyboard-enhancements",
		Short: "Keyboard enhancements and key release demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
