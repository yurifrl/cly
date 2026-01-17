package obsidian

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
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

func registerCapture(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "capture [text...]",
		Short: "Capture notes via Claude",
		Long:  "Runs claude -p '/capture <text>' headlessly in ~/Obsidian",
		RunE:  runCapture,
	}

	parent.AddCommand(cmd)
}

func runCapture(cmd *cobra.Command, args []string) error {
	prompt := "/capture " + strings.Join(args, " ")
	claudeArgs := []string{"-p", prompt}

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
