package helpy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const defaultFilePath = "~/DotFiles/HELP.md"

var fileFlag string

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

	// Create alias command
	alias := &cobra.Command{
		Use:   "hy",
		Short: "Alias for helpy",
		RunE:  run,
	}
	alias.Flags().StringVarP(&fileFlag, "file", "f", defaultFilePath, "path to help file")

	parent.AddCommand(cmd)
	parent.AddCommand(alias)
}

func run(cmd *cobra.Command, args []string) error {
	path := expandPath(fileFlag)

	if !fileExists(path) {
		msg := fmt.Sprintf("Help file not found: %s", path)
		fmt.Println(errorStyle.Render(msg))
		return nil
	}

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
