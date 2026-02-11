package aliases

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "aliases",
		Short: "Print shell aliases for cly commands",
		Long:  "Outputs alias commands you can eval in your shell. Skips names that conflict with existing binaries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := GenerateAliases(parent, exec.LookPath)
			fmt.Print(FormatFish(entries))
			return nil
		},
	}

	cmd.Flags().Bool("fish", false, "Output fish aliases (default)")

	parent.AddCommand(cmd)
}
