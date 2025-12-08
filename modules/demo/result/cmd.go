package result

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve a value from a Bubble Tea program",
		Long:  "Demonstrates how to retrieve a value from a Bubble Tea program after it exits",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())

	// Run returns the model as a tea.Model.
	m, err := p.Run()
	if err != nil {
		return err
	}

	// Assert the final tea.Model to our local model and print the choice.
	if m, ok := m.(model); ok && m.choice != "" {
		fmt.Printf("\n---\nYou chose %s!\n", m.choice)
	}
	return nil
}
