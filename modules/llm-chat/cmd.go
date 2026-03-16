package llmchat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "llm-chat",
		Short: "Interactive AI chat",
		Long: `Start an interactive chat session using direct API calls (Anthropic/OpenAI).

Examples:
  cly llm-chat                                    # Start new conversation (Anthropic)
  cly llm-chat --model claude-opus-4              # Use different model
  cly llm-chat --api openai --model gpt-4o        # Use OpenAI
`,
		RunE: run,
	}

	cmd.Flags().StringP("model", "m", "", "Model to use")
	cmd.Flags().StringP("api", "a", "anthropic", "API provider (anthropic, openai)")

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	flags := make(map[string]interface{})
	if model, _ := cmd.Flags().GetString("model"); model != "" {
		flags["model"] = model
	}
	if api, _ := cmd.Flags().GetString("api"); api != "" {
		flags["api"] = api
	}

	client, err := NewClient(flags)
	if err != nil {
		return err
	}

	conversationID := GenerateConversationID()
	fmt.Println(style.SubtleStyle.Render(conversationID))
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	isFirstMessage := true
	for {
		fmt.Print(style.YellowStyle.Render("> "))

		if !scanner.Scan() {
			break
		}

		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			break
		}

		ctx := context.Background()
		response, err := client.SendMessage(ctx, conversationID, prompt, isFirstMessage)
		if err != nil {
			fmt.Println(style.RedStyle.Render(fmt.Sprintf("Error: %s", err)))
			continue
		}

		fmt.Print(response)
		fmt.Println()

		isFirstMessage = false
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}
