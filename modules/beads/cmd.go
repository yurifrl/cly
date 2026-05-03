package beads

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

// Register attaches the beads command tree to parent.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "beads",
		Aliases: []string{"bd"},
		Short:   "Beads helpers",
		Long:    "Small TUIs for working with the beads (bd) issue tracker.",
		Annotations: map[string]string{
			// Don't emit shell aliases for `beads`/`bd` — the `bd` binary is
			// the real beads CLI and we don't want to shadow it.
			"cly.alias.skip": "true",
		},
	}

	newCmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"create", "n"},
		Short:   "Create a new bead (title, description, type)",
		Long:    "Compact inline form that shells out to `bd create` when you submit. Tab switches fields, → accepts a type suggestion, Ctrl+Enter submits, Esc cancels.",
		RunE:    runNew,
	}
	cmd.AddCommand(newCmd)

	// Make `cly beads` run the form directly for convenience.
	cmd.RunE = runNew

	parent.AddCommand(cmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())
	final, err := p.Run()
	if err != nil {
		return err
	}
	m, ok := final.(model)
	if !ok || !m.submitted {
		return nil
	}
	out, createErr := m.runCreate()
	if out != "" {
		fmt.Fprintln(cmd.OutOrStdout(), out)
	}
	return createErr
}
