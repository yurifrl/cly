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
)

const defaultFilePath = "~/DotFiles/HELP.md"

var fileFlag string
var ideFlag bool

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
	cmd.Flags().BoolVarP(&ideFlag, "ide", "i", false, "open Claude Code with IDE")

	// Create alias command
	alias := &cobra.Command{
		Use:   "hy",
		Short: "Alias for helpy",
		RunE:  run,
	}
	alias.Flags().StringVarP(&fileFlag, "file", "f", defaultFilePath, "path to help file")
	alias.Flags().BoolVarP(&ideFlag, "ide", "i", false, "open Claude Code with IDE")

	parent.AddCommand(cmd)
	parent.AddCommand(alias)
}

func run(cmd *cobra.Command, args []string) error {
	if ideFlag {
		return runClaude(cmd, args)
	}

	path := resolveFilePath(fileFlag)
	content, err := readFileContent(path)
	if err != nil {
		msg := fmt.Sprintf("Error reading file: %v", err)
		fmt.Println(errorStyle.Render(msg))
		return nil
	}

	m, err := initialModel(content)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func resolveFilePath(filePath string) string {
	if filePath == defaultFilePath {
		// Fast path - default file, skip config and dir scan
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "DotFiles", "HELP.md")
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

func runClaude(cmd *cobra.Command, args []string) error {
	dotfilesPath := expandPath("~/DotFiles")

	systemPrompt := "You are a question answerer for system configuration in ~/DotFiles. Your role is to explain configurations, find settings, and answer questions about the dotfiles, neovim config, shell configs, and system utilities. ONLY modify or change files if explicitly instructed. Default to reading and explaining, not editing."

	claudeArgs := []string{"--system-prompt", systemPrompt, "--ide"}
	claudeCmd := exec.Command("claude", claudeArgs...)
	claudeCmd.Dir = dotfilesPath
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}
