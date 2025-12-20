package ai

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
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
	// Get flags
	model, _ := cmd.Flags().GetString("model")
	conversationID, _ := cmd.Flags().GetString("continue")

	// Create client (mods will use its own config for API key)
	client, err := NewClient("", model)
	if err != nil {
		return err
	}

	// Generate conversation ID if not continuing
	if conversationID == "" {
		conversationID = generateConversationID()
	}

	// Welcome message
	fmt.Println(style.SubtleStyle.Render(fmt.Sprintf("Conversation: %s", conversationID)))
	fmt.Println()

	// Chat loop
	scanner := bufio.NewScanner(os.Stdin)
	isFirstMessage := true
	for {
		// Prompt
		fmt.Print(style.YellowStyle.Render("> "))

		// Read input
		if !scanner.Scan() {
			break
		}

		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			break
		}

		// Send to mods
		ctx := context.Background()
		response, err := client.SendMessage(ctx, conversationID, prompt, isFirstMessage)
		if err != nil {
			fmt.Println(style.RedStyle.Render(fmt.Sprintf("Error: %s", err)))
			continue
		}

		// Print response
		fmt.Print(response)
		fmt.Println()

		// After first message, switch to continue mode
		isFirstMessage = false
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}
