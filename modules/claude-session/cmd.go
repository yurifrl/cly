package claudesession

import (
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

func execClaude(entry *Entry) error {
	return session.ExecClaude([]string{"-r", entry.ID})
}

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "claude-sessions",
		Aliases: []string{"cs"},
		Short:   "Manage Claude sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLS(cmd, false)
		},
	}

	cmd.AddCommand(lsCmd())
	cmd.AddCommand(rmCmd())
	cmd.AddCommand(saveCmd())
	cmd.AddCommand(resumeCmd())

	parent.AddCommand(cmd)
}
