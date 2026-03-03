package llmchat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
)

// Register registers the llm-chat command with the parent command
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "llm-chat",
		Short: "Interactive AI chat powered by mods",
		Long: `Start an interactive chat session with Claude using mods.

The conversation is threaded and stored in mods' cache (~/.cache/mods/).
You can continue conversations with mods CLI using the same conversation ID.

Examples:
  cly llm-chat                             # Start new conversation
  cly llm-chat --model claude-opus-4       # Use different model
`,
		RunE: run,
	}

	cmd.Flags().StringP("model", "m", "", "Model to use")
	cmd.Flags().StringP("continue", "c", "", "Continue conversation by ID")
	cmd.Flags().BoolP("continue-last", "C", false, "Continue from last conversation")
	cmd.Flags().BoolP("list", "l", false, "List saved conversations")
	cmd.Flags().StringP("show", "s", "", "Show conversation by ID")
	cmd.Flags().BoolP("show-last", "S", false, "Show last conversation")
	cmd.Flags().Bool("settings", false, "Open settings in $EDITOR")
	cmd.Flags().Bool("mcp-list", false, "List all available MCP servers")
	cmd.Flags().Bool("mcp-list-tools", false, "List all available tools from enabled MCP servers")
	cmd.Flags().String("mcp-disable", "", "Disable specific MCP servers")

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	// Check for non-interactive commands first
	if settings, _ := cmd.Flags().GetBool("settings"); settings {
		return runMods("--settings")
	}
	if mcpList, _ := cmd.Flags().GetBool("mcp-list"); mcpList {
		return runMods("--mcp-list")
	}
	if mcpListTools, _ := cmd.Flags().GetBool("mcp-list-tools"); mcpListTools {
		return runMods("--mcp-list-tools")
	}
	if mcpDisable, _ := cmd.Flags().GetString("mcp-disable"); mcpDisable != "" {
		return runMods("--mcp-disable", mcpDisable)
	}
	if list, _ := cmd.Flags().GetBool("list"); list {
		return runMods("-l")
	}
	if show, _ := cmd.Flags().GetString("show"); show != "" {
		return runMods("-s", show)
	}
	if showLast, _ := cmd.Flags().GetBool("show-last"); showLast {
		return runMods("-S")
	}

	// Get flags
	conversationID, _ := cmd.Flags().GetString("continue")
	continueLast, _ := cmd.Flags().GetBool("continue-last")

	// Collect mods flags
	modsFlags := make(map[string]interface{})
	if model, _ := cmd.Flags().GetString("model"); model != "" {
		modsFlags["model"] = model
	}
	if continueLast {
		modsFlags["continue-last"] = true
	}

	// Create client
	client, err := NewClient(modsFlags)
	if err != nil {
		return err
	}

	// Generate conversation ID if not continuing
	isContinuing := conversationID != ""
	if conversationID == "" {
		conversationID = GenerateConversationID()
	}

	// Print conversation ID
	fmt.Println(style.SubtleStyle.Render(conversationID))
	fmt.Println()

	// Chat loop
	scanner := bufio.NewScanner(os.Stdin)
	isFirstMessage := !isContinuing
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

// runMods executes mods with given arguments and streams output
func runMods(args ...string) error {
	cmd := exec.Command("mods", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
