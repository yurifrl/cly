package claudesession

import (
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

// execClaudeArgs builds the args for resuming a session, with optional yolo mode.
func execClaudeArgs(entry *Entry, yolo bool) []string {
	args := []string{"-r", entry.ID}
	if yolo {
		return append(session.YoloArgs(), args...)
	}
	return args
}

func execClaude(entry *Entry, yolo bool) error {
	sess := &session.Session{Name: entry.Name}
	_ = sess.RenameZellijTab()
	return sess.ExecClaude(execClaudeArgs(entry, yolo))
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
