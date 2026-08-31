package omp

import (
	"github.com/spf13/cobra"
)

// Register attaches the `extensions` command to parent.
func Register(parent *cobra.Command) {
	ext := &cobra.Command{
		Use:   "extensions",
		Short: "Manage omp extensions shipped by cly",
	}
	ext.AddCommand(installCmd())

	parent.AddCommand(ext)
}
