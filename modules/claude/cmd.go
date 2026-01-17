package claude

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:                "claude",
		Short:              "Run Claude Code with session management",
		Long:               "Wraps Claude Code with session naming and Zellij tab integration",
		DisableFlagParsing: true,
		RunE:               run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	var name string
	var anonymous bool
	var passArgs []string

	// Parse flags manually since we disabled flag parsing
	for i := 0; i < len(args); i++ {
		if args[i] == "--name" || args[i] == "-n" {
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				name = args[i+1]
				i++
			}
		} else if args[i] == "--anonymous" || args[i] == "-a" {
			anonymous = true
		} else {
			passArgs = append(passArgs, args[i])
		}
	}

	if anonymous {
		fmt.Println("🥷 Anonymous mode")
		return session.ExecClaudeAnonymous(passArgs)
	}

	sess, err := session.Initialize(name)
	if err != nil {
		return err
	}

	fmt.Printf("🏷️  Session: %s\n", sess.Name)

	_ = sess.RenameZellijTab()

	return sess.ExecClaude(passArgs)
}
