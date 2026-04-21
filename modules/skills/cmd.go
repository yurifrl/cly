// Package skills installs cly-bundled AI agent skills to the local filesystem.
package skills

import (
	"github.com/spf13/cobra"
)

// Register attaches the `skills` command tree to parent.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage cly-bundled AI agent skills",
		Long:  "Install cly-bundled AI agent skills (SKILL.md files) into a local skills directory.",
	}
	cmd.AddCommand(installCmd())
	parent.AddCommand(cmd)
}
