// Package pi installs cly-bundled pi TUI extensions to the local pi extensions dir.
package pi

import (
	"github.com/spf13/cobra"
)

// Register attaches the `extensions` command to parent.
func Register(parent *cobra.Command) {
	ext := &cobra.Command{
		Use:   "extensions",
		Short: "Manage pi extensions shipped by cly",
	}
	ext.AddCommand(installCmd())

	parent.AddCommand(ext)
}
