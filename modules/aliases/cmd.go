package aliases

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/completion"
)

func Register(parent *cobra.Command) {
	// Register lazy completion generator — runs when `cly completion fish` executes,
	// after all modules are registered on parent.
	completion.RegisterLazy(func() string {
		entries := GenerateAliases(parent, exec.LookPath)
		skip := completion.RegisteredAliases()
		return FormatFishCompletions(entries, skip...)
	})

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
