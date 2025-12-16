package ai

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Register registers the ai command with the parent command
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "Interactive AI chat powered by mods",
		Long: `Start an interactive chat session with Claude using mods.

The conversation is threaded and stored in mods' cache (~/.cache/mods/).
You can continue conversations with mods CLI using the same conversation ID.

Examples:
  cly ai                                    # Start new conversation
  cly ai --model claude-opus-4            # Use different model
`,
		RunE: run,
	}

	cmd.Flags().String("model", "", "Model to use (default: claude-3-5-sonnet-latest)")
	cmd.Flags().String("continue", "", "Continue existing conversation by ID")

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Get model from flag
	model, _ := cmd.Flags().GetString("model")

	// Create model
	m, err := NewModel(apiKey, model)
	if err != nil {
		return err
	}

	// Check for continue flag
	continueID, _ := cmd.Flags().GetString("continue")
	if continueID != "" {
		m.conversationID = continueID
	}

	// Run TUI
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
