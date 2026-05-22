package piwrap

import (
	"os"

	"github.com/spf13/cobra"
)

// Register attaches `cly pi` (alias `p`) to parent. Flag parsing is
// disabled so all flags pass through to the underlying pi binary,
// except --name/-n which is intercepted in Run.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:                "pi [-n NAME] [pi args...]",
		Aliases:            []string{"p"},
		Short:              "Wrap pi with --name to label the session and rename the cmux tab",
		Long:               "Thin pass-through to the `pi` binary.\n\nAdds --name / -n: sets $CLY_SESSION_NAME for the pi process and renames the current cmux tab. All other arguments are forwarded to pi unchanged.",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := Run(args)
			if err != nil {
				// Propagate pi's exit code without cobra adding noise.
				if exitErr, ok := err.(interface{ ExitCode() int }); ok {
					os.Exit(exitErr.ExitCode())
				}
			}
			return err
		},
	}
	parent.AddCommand(cmd)
}
