package helpy

import (
	"github.com/yurifrl/cly/pkg/ai"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/llm"
	"golang.org/x/term"
)

const defaultFilePath = "~/DotFiles/HELP.md"
const defaultDocsDir = "~/DotFiles/docs"

var fileFlag string
var ideFlag string
var promptFlag string
var docsFlag bool
var chatFlag bool
var interactiveFlag bool
var outputFlag string

// addFlags registers the shared flags on a command.
func addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&fileFlag, "file", "f", defaultFilePath, "path to help file")
	cmd.Flags().StringVarP(&ideFlag, "ide", "i", "", "open IDE tool (pi, claude, gemini, etc.) [$CLY_HELPY_IDE]")
	cmd.Flags().Lookup("ide").NoOptDefVal = ideDefault()
	cmd.Flags().StringVarP(&promptFlag, "prompt", "p", "", "send prompt to Claude")
	cmd.Flags().BoolVarP(&docsFlag, "docs", "d", false, "browse docs with fuzzy finder")
	cmd.Flags().BoolVarP(&chatFlag, "chat", "c", false, "open AI chat with doc context")
	cmd.Flags().BoolVar(&interactiveFlag, "it", false, "open interactive TUI viewer")
	cmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output format: json, raw")
	_ = cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json\tStructured JSON output", "raw\tRaw markdown (no rendering)"}, cobra.ShellCompDirectiveNoFileComp
	})
}

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
		Use:               "helpy [section]",
		Short:             "Display help file in a TUI viewer",
		Long:              "Display a markdown help file with syntax highlighting and scrolling.\nOptionally pass a section name to jump directly to that header.",
		RunE:              run,
		ValidArgsFunction: headerCompletions,
	}
	addFlags(cmd)

	sectionsCmd := &cobra.Command{
		Use:   "sections",
		Short: "List available sections",
		RunE:  runSections,
	}
	sectionsCmd.Flags().StringVarP(&fileFlag, "file", "f", defaultFilePath, "path to help file")
	sectionsCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output format: json")
	_ = sectionsCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json\tStructured JSON output"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.AddCommand(sectionsCmd)

	alias := &cobra.Command{
		Use:               "hy [section]",
		Short:             "Alias for helpy",
		RunE:              run,
		ValidArgsFunction: headerCompletions,
	}
	addFlags(alias)

	hAlias := &cobra.Command{
		Use:               "h [section]",
		Short:             "Alias for helpy",
		RunE:              run,
		ValidArgsFunction: headerCompletions,
	}
	addFlags(hAlias)

	parent.AddCommand(cmd)
	parent.AddCommand(alias)
	parent.AddCommand(hAlias)
}

// headerCompletions returns HELP.md headers as slug completions.
func headerCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete the first arg (the section to jump to)
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path := resolveFilePath(fileFlag)
	content, err := readFileContent(path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	_, body := parseFrontmatter(content)
	headers := extractHeaders(body)

	var completions []string
	for _, h := range headers {
		completions = append(completions, h.slug+"\t"+h.title)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func run(cmd *cobra.Command, args []string) error {
	if docsFlag {
		return runDocsPicker()
	}
	if chatFlag {
		return runChat()
	}
	if promptFlag != "" {
		return runPrompt(promptFlag)
	}
	if ideFlag != "" {
		return runIDE(cmd, args, ideFlag)
	}

	path := resolveFilePath(fileFlag)
	rawContent, err := readFileContent(path)
	if err != nil {
		if outputFlag == "json" {
			return jsonError("file_not_found", fmt.Sprintf("Error reading file: %v", err))
		}
		msg := fmt.Sprintf("Error reading file: %v", err)
		fmt.Println(errorStyle.Render(msg))
		return nil
	}

	meta, content := parseFrontmatter(rawContent)

	// If a section slug was provided, extract only that section
	if len(args) > 0 {
		slug := args[0]
		section, found := extractSection(content, slug)
		if found {
			content = section
		} else if outputFlag == "json" {
			return jsonError("section_not_found", fmt.Sprintf("No section matching slug %q", slug))
		}
	}

	// Output mode selection
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	if outputFlag == "json" {
		return outputJSON(content, meta, args)
	}
	if outputFlag == "raw" {
		fmt.Print(content)
		return nil
	}
	if !isTTY {
		return outputJSON(content, meta, args)
	}

	// --it opens the interactive TUI, otherwise render to stdout
	if !interactiveFlag {
		return renderToStdout(content)
	}

	// Interactive TUI
	m, err := initialModel(content)
	if err != nil {
		return err
	}

	client, clientErr := helpyClient()
	if clientErr == nil && client != nil {
		m.chat = newChatModel(client, helpySystemPrompt(), content, meta)
		m.chatEnabled = true
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// renderToStdout renders markdown with glamour and prints to stdout.
func renderToStdout(content string) error {
	r, err := glamour.NewTermRenderer(glamour.WithEnvironmentConfig(), glamour.WithWordWrap(80))
	if err != nil {
		fmt.Print(content)
		return nil
	}
	out, err := r.Render(content)
	if err != nil {
		fmt.Print(content)
		return nil
	}
	fmt.Print(out)
	return nil
}

// sectionJSON is the structured output for --output json.
type sectionJSON struct {
	Content  string        `json:"content"`
	Sections []sectionMeta `json:"sections"`
	File     string        `json:"file"`
	Section  string        `json:"section,omitempty"`
}

type sectionMeta struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Level int    `json:"level"`
}

func outputJSON(content string, meta DocMeta, args []string) error {
	headers := extractHeaders(content)
	sections := make([]sectionMeta, len(headers))
	for i, h := range headers {
		sections[i] = sectionMeta{Slug: h.slug, Title: h.title, Level: h.level}
	}

	out := sectionJSON{
		Content:  content,
		Sections: sections,
		File:     resolveFilePath(fileFlag),
	}
	if len(args) > 0 {
		out.Section = args[0]
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func jsonError(code, message string) error {
	out := map[string]string{"error": code, "message": message}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
	return fmt.Errorf("%s: %s", code, message)
}

// runSections lists all available sections.
func runSections(cmd *cobra.Command, args []string) error {
	path := resolveFilePath(fileFlag)
	rawContent, err := readFileContent(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	_, content := parseFrontmatter(rawContent)
	headers := extractHeaders(content)

	if outputFlag == "json" {
		sections := make([]sectionMeta, len(headers))
		for i, h := range headers {
			sections[i] = sectionMeta{Slug: h.slug, Title: h.title, Level: h.level}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sections)
	}

	for _, h := range headers {
		indent := strings.Repeat("  ", h.level-1)
		fmt.Printf("%s%s  %s\n", indent, h.slug, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(h.title))
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
	client, err := helpyClient()
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to create AI client: %v", err)))
		return nil
	}
	if client == nil {
		fmt.Println(errorStyle.Render("AI chat is disabled in config"))
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

	chat := newStandaloneChatModel(client, helpySystemPrompt(), docContent, meta)
	p := tea.NewProgram(chat)
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
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// helpyClient builds the LLM client from global `ai:` config plus helpy's
// own override under `modules.helpy.ai`.
func helpyClient() (llm.Client, error) {
	return ai.NewClientFor("helpy")
}

// helpySystemPrompt returns the system prompt configured under
// `modules.helpy.ai.system_prompt`, or a sensible default. Per-module
// prompts intentionally live with their module, not in `pkg/ai`.
func helpySystemPrompt() string {
	defaultPrompt := "You are a helpful assistant. Answer questions about the document provided. Be concise and reference specific sections when possible."
	o := ai.LookupModuleOverride("helpy")
	if o == nil {
		return defaultPrompt
	}
	if v, ok := o["system_prompt"].(string); ok && v != "" {
		return v
	}
	return defaultPrompt
}
