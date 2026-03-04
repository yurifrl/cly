package lg_color_dialog

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-color-dialog command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-color-dialog",
		Short: "Lipgloss color dialog demo",
		Long:  "A banana-themed dialog box with rounded purple border, paragraph text, and Yes/No buttons",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderDialog())
	return nil
}
