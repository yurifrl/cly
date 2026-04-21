// Package pi installs cly-bundled pi TUI extensions to the local pi extensions dir.
package pi

import (
	"github.com/spf13/cobra"
)

// Register attaches the `pi` command tree to parent.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "pi",
		Short: "Manage cly-bundled pi (coding agent) integrations",
		Long:  "Install cly-bundled pi extensions into the local pi extensions directory.",
	}

	ext := &cobra.Command{
		Use:   "extensions",
		Short: "Manage pi extensions shipped by cly",
	}
	ext.AddCommand(installCmd())
	cmd.AddCommand(ext)

	parent.AddCommand(cmd)
}
