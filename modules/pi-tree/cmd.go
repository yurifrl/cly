package pitree

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the pi-tree command to parent.
func Register(parent *cobra.Command) {
	var flagJSON  bool
	var flagSave  bool
	var flagSince float64

	cmd := &cobra.Command{
		Use:     "pi-tree",
		Aliases: []string{"pt", "pitree"},
		Short:   "Show open cmux workspaces and their π sessions as a tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI2(cmd, flagJSON, flagSave, flagSince)
		},
	}
	cmd.Flags().BoolVar(&flagJSON, "json", false, "Output current tree as JSON")
	cmd.Flags().BoolVar(&flagSave, "save", false, "Force save a new snapshot version")
	cmd.Flags().Float64Var(&flagSince, "since", 0, "Only show sessions started within the last N hours (e.g. 24, 0.5)")

	cmd.AddCommand(historyCmd())

	parent.AddCommand(cmd)
}

func runTUI2(cmd *cobra.Command, jsonOut bool, forceSave bool, sinceHours float64) error {
	nodes, err := ScanTree()
	if err != nil {
		// Non-fatal — we may have snapshots to show
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: scan: %v\n", err)
	}

	if jsonOut {
		data, err := MarshalJSON(nodes)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Always update latest snapshot in-place with current live data.
	// Only --save creates a NEW version.
	if len(nodes) > 0 {
		_, _, err := Upsert(nodes, forceSave)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not save snapshot: %v\n", err)
		}
	}

	all, _ := LoadSnapshots()
	snapshots := ActiveSnapshots(all)
	return RunTUI(nodes, snapshots, sinceHours)
}

func historyCmd() *cobra.Command {
	var flagJSON bool
	cmd := &cobra.Command{
		Use:   "history",
		Short: "List saved pi-tree snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := LoadSnapshots()
			if err != nil {
				return err
			}
			snapshots := ActiveSnapshots(all)
			if len(snapshots) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No snapshots yet. Run `cly pi-tree` first.")
				return nil
			}

			if flagJSON {
				data, err := MarshalJSON(snapshots)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "  %-4s  %-16s  %-16s  %s\n",
				"ver", "created", "updated", "sessions")
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", "────────────────────────────────────────────────────")
			for _, s := range snapshots {
				count := 0
				for _, ws := range s.Tree {
					count += len(ws.Sessions)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  v%-3d  %s  %s  %d\n",
					s.Version,
					s.CreatedAt.Format("2006-01-02 15:04"),
					s.UpdatedAt.Format("2006-01-02 15:04"),
					count,
				)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagJSON, "json", false, "Output snapshots as JSON")
	return cmd
}

// MarshalJSON serializes any value to indented JSON.
func MarshalJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
