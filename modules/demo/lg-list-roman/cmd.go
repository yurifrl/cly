package lg_list_roman

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-list-roman command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-list-roman",
		Short: "Lipgloss list: Roman numeral enumerated makeup brands",
		Long:  "A styled list of makeup brands using Roman numeral enumerators with colored enumerator and item styles",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderRomanList())
	return nil
}
