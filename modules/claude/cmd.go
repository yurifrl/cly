package claude

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	claudesession "github.com/yurifrl/cly/modules/claude-session"
	"github.com/yurifrl/cly/pkg/session"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:                "claude",
		Aliases:            []string{"c"},
		Short:              "Run Claude Code with session management",
		Long:               "Wraps Claude Code with session naming and Zellij tab integration",
		DisableFlagParsing: true,
		RunE:               run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := ParseArgs(args)

	if p.Anonymous {
		fmt.Println("🥷 Anonymous mode")
		return session.ExecClaudeAnonymous(p.PassArgs)
	}

	if p.ContinueSession != "" {
		return restoreSession(p.ContinueSession)
	}

	if p.SessionName != "" {
		return resumeOrCreateSession(p.SessionName)
	}

	sess, err := session.Initialize(p.Name)
	if err != nil {
		return err
	}

	fmt.Printf("🏷️  Session: %s\n", sess.Name)

	_ = sess.RenameZellijTab()

	return sess.ExecClaude(p.PassArgs, session.WithTaskListID(p.TaskListID))
}

func restoreSession(name string) error {
	sessions, err := claudesession.Load(claudesession.FilePath())
	if err != nil {
		return err
	}

	entry := claudesession.FindByName(sessions, name)
	if entry == nil {
		return fmt.Errorf("session %q not found", name)
	}

	fmt.Printf("Resuming session: %s\n", name)

	if err := os.Chdir(entry.Path); err != nil {
		return fmt.Errorf("chdir %s: %w", entry.Path, err)
	}

	return session.ExecClaude([]string{"-r", entry.ID})
}

func resumeOrCreateSession(name string) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	sessions, err := claudesession.Load(claudesession.FilePath())
	if err != nil {
		return err
	}

	// Try to find existing session in current directory
	entry := claudesession.FindByNameAndPath(sessions, name, path)
	if entry != nil {
		fmt.Printf("📂 Resuming session: %s (id=%s)\n", name, entry.ID)
		return session.ExecClaude([]string{"-r", entry.ID})
	}

	// Create new session
	fmt.Printf("✨ Creating new session: %s\n", name)

	sessionID := uuid.New().String()

	// Save to sessions.json
	key := claudesession.MakeKey(path, sessionID)
	sessions[key] = claudesession.Entry{
		ID:   sessionID,
		Name: name,
		Path: path,
	}
	if err := claudesession.Save(claudesession.FilePath(), sessions); err != nil {
		return err
	}

	return session.ExecClaude([]string{"--session-id", sessionID})
}
