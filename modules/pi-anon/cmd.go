package pianon

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "anon [session-words]",
		Short: "Anonymous pi sessions in /tmp",
		Long: `Create or resume anonymous pi sessions.

New session:    cly py anon
Resume:         cly py anon monkey-car-whale

Sessions live in /tmp/<timestamp>-<words>-pi-anon/ with a .anon.json
tracking session ID, timestamps, toggle preferences, and calling dirs.`,
		Args: cobra.MaximumNArgs(1),
		RunE: run,
	}

	cmd.Flags().Bool("list", false, "List all existing sessions")

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	listFlag, _ := cmd.Flags().GetBool("list")

	if listFlag {
		return listSessions()
	}

	if len(args) > 0 {
		return resumeSession(args[0])
	}

	return newSession()
}

func listSessions() error {
	sessions, err := findAllSessions()
	if err != nil || len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("📋 Anonymous sessions:")
	for _, s := range sessions {
		fmt.Printf("   %s  →  %s\n", s.Words, s.Dir)
	}
	return nil
}
