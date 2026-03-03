package obsidian

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	checkMark = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			Render("✓")

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)
)

func buildClaudeArgs(args []string) []string {
	prompt := "/capture " + strings.Join(args, " ")
	return []string{"-p", prompt, "--allowedTools", "WebFetch,WebSearch"}
}

func runCapture(args []string) error {
	claudeArgs := buildClaudeArgs(args)

	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Obsidian")

	fmt.Println(style.SubtleStyle.Render("Running in ") + pathStyle.Render(dir))
	fmt.Println()

	c := exec.Command("claude", claudeArgs...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	err := c.Run()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(checkMark + " " + style.GreenStyle.Render("Captured to ") + pathStyle.Render(dir))

	return nil
}
