package helpy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/llm"
)

const defaultFilePath = "~/DotFiles/HELP.md"
const defaultDocsDir = "~/DotFiles/docs"

var fileFlag string
var ideFlag string
var promptFlag string
var docsFlag bool
var chatFlag bool

// ideDefault returns the default IDE tool name from env or "pi".
func ideDefault() string {
	if v := os.Getenv("CLY_HELPY_IDE"); v != "" {
		return v
	}
	return "pi"
}

var errorStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("208")).
	Foreground(lipgloss.Color("208")).
	Padding(1, 2).
	Margin(1)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "helpy",
		Short: "Display help file in a TUI viewer",
		Long:  "Display a markdown help file with syntax highlighting and scrolling",
		RunE:  run,
	}

	cmd.Flags().StringVarP(&fileFlag, "file", "f", defaultFilePath, "path to help file")
	cmd.Flags().StringVarP(&ideFlag, "ide", "i", "", "open IDE tool (pi, claude, gemini, etc.) [$CLY_HELPY_IDE]")
	cmd.Flags().Lookup("ide").NoOptDefVal = ideDefault()
	cmd.Flags().StringVarP(&promptFlag, "prompt", "p", "", "send prompt to Claude")
	cmd.Flags().BoolVarP(&docsFlag, "docs", "d", false, "browse docs with fuzzy finder")
	cmd.Flags().BoolVarP(&chatFlag, "chat", "c", false, "open AI chat with doc context")

	// Create alias command
	alias := &cobra.Command{
		Use:   "hy",
		Short: "Alias for helpy",
		RunE:  run,
	}
	alias.Flags().StringVarP(&fileFlag, "file", "f", defaultFilePath, "path to help file")
	alias.Flags().StringVarP(&ideFlag, "ide", "i", "", "open IDE tool (pi, claude, gemini, etc.) [$CLY_HELPY_IDE]")
	alias.Flags().Lookup("ide").NoOptDefVal = ideDefault()
	alias.Flags().StringVarP(&promptFlag, "prompt", "p", "", "send prompt to Claude")
	alias.Flags().BoolVarP(&docsFlag, "docs", "d", false, "browse docs with fuzzy finder")
	alias.Flags().BoolVarP(&chatFlag, "chat", "c", false, "open AI chat with doc context")

	parent.AddCommand(cmd)
	parent.AddCommand(alias)
}

func run(cmd *cobra.Command, args []string) error {
	// Handle --docs flag
	if docsFlag {
		return runDocsPicker()
	}

	// Handle --chat flag: standalone AI chat with doc context
	if chatFlag {
		return runChat()
	}

	// Handle --prompt flag
	if promptFlag != "" {
		return runPrompt(promptFlag)
	}

	if ideFlag != "" {
		return runIDE(cmd, args, ideFlag)
	}

	path := resolveFilePath(fileFlag)
	rawContent, err := readFileContent(path)
	if err != nil {
		msg := fmt.Sprintf("Error reading file: %v", err)
		fmt.Println(errorStyle.Render(msg))
		return nil
	}

	meta, content := parseFrontmatter(rawContent)

	m, err := initialModel(content)
	if err != nil {
		return err
	}

	// Set up AI chat if configured
	aiCfg := loadAIConfig()
	if aiCfg != nil {
		client, clientErr := llm.NewClient(*aiCfg)
		if clientErr == nil {
			m.chat = newChatModel(client, aiCfg.SystemPrompt, content, meta)
			m.chatEnabled = true
		}
		// Silently skip AI if client creation fails (e.g., no API key)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func resolveFilePath(filePath string) string {
	if filePath == defaultFilePath {
		// Fast path - default file, use config
		cfg := config.Get()
		dotfilesPath := expandPath(cfg.App.DotFilesDir)
		return filepath.Join(dotfilesPath, "HELP.md")
	}
	// User-specified file - just expand ~
	return expandPath(filePath)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func runIDE(cmd *cobra.Command, args []string, tool string) error {
	cfg := config.Get()
	dotfilesPath := expandPath(cfg.App.DotFilesDir)
	helpFile := filepath.Join(dotfilesPath, "HELP.md")

	var promptBuilder strings.Builder

	// Get preprompt from config
	if helpyConfig, ok := cfg.Modules["helpy"]; ok {
		if preprompt, ok := helpyConfig["preprompt"].(string); ok && preprompt != "" {
			promptBuilder.WriteString(preprompt)
			promptBuilder.WriteString("\n\n")
		}
	}

	// Add DotFiles HELP.md content
	if content, err := readFileContent(helpFile); err == nil {
		promptBuilder.WriteString(content)
	}

	var ideCmd *exec.Cmd
	switch strings.ToLower(tool) {
	case "claude":
		claudeArgs := []string{}
		if promptBuilder.Len() > 0 {
			claudeArgs = append(claudeArgs, "--system-prompt", promptBuilder.String())
		}
		claudeArgs = append(claudeArgs, "--ide")
		ideCmd = exec.Command("claude", claudeArgs...)

	case "pi":
		piArgs := []string{}
		if promptBuilder.Len() > 0 {
			piArgs = append(piArgs, "--system-prompt", promptBuilder.String())
		}
		ideCmd = exec.Command("pi", piArgs...)

	default:
		// Generic: just run the tool name as a command
		ideCmd = exec.Command(tool)
	}

	ideCmd.Dir = dotfilesPath
	ideCmd.Stdin = os.Stdin
	ideCmd.Stdout = os.Stdout
	ideCmd.Stderr = os.Stderr

	return ideCmd.Run()
}

func runPrompt(prompt string) error {
	cfg := config.Get()
	dotfilesPath := expandPath(cfg.App.DotFilesDir)

	claudeArgs := []string{"-p", prompt}

	claudeCmd := exec.Command("claude", claudeArgs...)
	claudeCmd.Dir = dotfilesPath
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}

// runChat opens a full-screen interactive AI chat TUI with the current doc as context.
func runChat() error {
	aiCfg := loadAIConfig()
	if aiCfg == nil {
		fmt.Println(errorStyle.Render("AI chat is disabled in config"))
		return nil
	}

	client, err := llm.NewClient(*aiCfg)
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to create AI client: %v", err)))
		return nil
	}

	// Load the doc for context
	path := resolveFilePath(fileFlag)
	rawContent, readErr := readFileContent(path)

	var docContent string
	var meta DocMeta
	if readErr == nil {
		meta, docContent = parseFrontmatter(rawContent)
	}

	chat := newStandaloneChatModel(client, aiCfg.SystemPrompt, docContent, meta)
	p := tea.NewProgram(chat, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func runDocsPicker() error {
	cfg := config.Get()

	// Get docs_dir from config, fallback to default
	docsDir := defaultDocsDir
	if helpyConfig, ok := cfg.Modules["helpy"]; ok {
		if dir, ok := helpyConfig["docs_dir"].(string); ok && dir != "" {
			docsDir = dir
		}
	}

	docs, err := discoverDocs(docsDir)
	if err != nil {
		msg := fmt.Sprintf("Error discovering docs: %v", err)
		fmt.Println(errorStyle.Render(msg))
		return nil
	}

	m := newPickerModel(docs)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// loadAIConfig reads the AI configuration from helpy config.
// Falls back to sensible defaults if the user config doesn't have the ai section.
func loadAIConfig() *llm.Config {
	cfg := config.Get()

	// Defaults
	provider := "anthropic"
	model := "claude-sonnet-4-20250514"
	apiKey := ""
	apiKeyEnv := "ANTHROPIC_API_KEY"
	systemPrompt := "You are a helpful assistant. Answer questions about the document provided. Be concise and reference specific sections when possible."
	enabled := true

	helpyConfig, ok := cfg.Modules["helpy"]
	if ok {
		if aiRaw, ok := helpyConfig["ai"]; ok {
			if aiMap, ok := aiRaw.(map[string]interface{}); ok {
				if v, ok := aiMap["enabled"].(bool); ok {
					enabled = v
				}
				if v, ok := aiMap["provider"].(string); ok && v != "" {
					provider = v
				}
				if v, ok := aiMap["model"].(string); ok && v != "" {
					model = v
				}
				if v, ok := aiMap["api_key"].(string); ok && v != "" {
					apiKey = v
				}
				if v, ok := aiMap["api_key_env"].(string); ok && v != "" {
					apiKeyEnv = v
				}
				if v, ok := aiMap["system_prompt"].(string); ok && v != "" {
					systemPrompt = v
				}
			}
		}
	}

	if !enabled {
		return nil
	}

	return &llm.Config{
		Provider:     llm.Provider(provider),
		Model:        model,
		APIKey:       apiKey,
		APIKeyEnv:    apiKeyEnv,
		SystemPrompt: systemPrompt,
	}
}
