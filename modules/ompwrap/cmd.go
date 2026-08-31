package ompwrap

import (
	"os"

	"github.com/spf13/cobra"
	clyomp "github.com/yurifrl/cly/modules/omp"
)
// Register attaches `cly omp` (alias `o`) to parent. Flag parsing is
// disabled so all flags pass through to the underlying omp binary,
// except --name/-n which is intercepted in Run.
//
// The command is named `omp` so `cly omp` reads naturally, but the shell
// alias generator skips the `omp` name (a real omp binary exists on PATH)
// and only emits the cobra alias `o` — mirroring how `pi`/`p` work.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:                "omp [-n NAME] [omp args...]",
		Aliases:            []string{"o"},
		Short:              "Wrap omp with --name to label the session and rename the cmux tab",
		Long:               "Thin pass-through to the `omp` binary.\n\nAdds --name / -n: sets $CLY_SESSION_NAME for the omp process and renames the current cmux tab. All other arguments are forwarded to omp unchanged.",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := Run(args)
			if err != nil {
				// Propagate omp's exit code without cobra adding noise.
				if exitErr, ok := err.(interface{ ExitCode() int }); ok {
					os.Exit(exitErr.ExitCode())
				}
			}
			return err
		},
	}

	// `omp y` namespaces cly's own omp tooling (not forwarded to the omp binary).
	y := &cobra.Command{
		Use:   "y",
		Short: "cly-owned omp coding agent tools",
		Long:  "Namespace for cly's own omp helpers. Everything under `omp y` is a cly command (not forwarded to the omp binary).",
	}
	clyomp.Register(y)
	cmd.AddCommand(y)

	parent.AddCommand(cmd)
}
