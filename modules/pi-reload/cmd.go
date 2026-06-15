// Package pireload broadcasts a slash command (default /reload) to every
// pi surface managed by cmux.
//
// Pi surfaces are identified by title regex (default matches the "π - "
// prefix set by the custom-footer extension). Use --match to override.
package pireload

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

type surface struct {
	Ref   string `json:"ref"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type cmuxTree struct {
	Windows []struct {
		Workspaces []struct {
			Panes []struct {
				Surfaces []surface `json:"surfaces"`
			} `json:"panes"`
		} `json:"workspaces"`
	} `json:"windows"`
}

// Register attaches the `reload` subcommand to parent.
func Register(parent *cobra.Command) {
	var (
		match        string
		listOnly     bool
		cmdName      string
		allTerminals bool
		yes          bool
	)

	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Broadcast /reload (or any slash command) to all running pi surfaces via cmux",
		Long: `Broadcast a slash command to every pi surface cmux knows about.

By default targets surfaces whose title starts with "π - " (set by the
custom-footer extension). Override with --match to include custom-titled
surfaces (e.g. "memwatch", "idle"), or use --all-terminals to hit every
terminal surface (dangerous — will send the command to non-pi shells too).

Examples:
  cly pi y reload --list
  cly pi y reload --match '^π -|^memwatch$|^idle$'
  cly pi y reload --cmd idle
  cly pi y reload -y`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd, match, listOnly, cmdName, allTerminals, yes)
		},
	}

	cmd.Flags().StringVar(&match, "match", `^π - `, "Regex applied to surface title to select targets")
	cmd.Flags().BoolVar(&listOnly, "list", false, "Show matched surfaces without sending")
	cmd.Flags().StringVar(&cmdName, "cmd", "reload", "Slash command to send (without the /)")
	cmd.Flags().BoolVar(&allTerminals, "all-terminals", false, "Match every terminal surface (dangerous)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(cmd)
}

func run(cobraCmd *cobra.Command, match string, listOnly bool, cmdName string, allTerminals bool, yes bool) error {
	stdout := cobraCmd.OutOrStdout()
	stderr := cobraCmd.ErrOrStderr()

	raw, err := exec.Command("cmux", "tree", "--all", "--json").Output()
	if err != nil {
		return fmt.Errorf("cmux tree --all --json failed: %w", err)
	}

	var tree cmuxTree
	if err := json.Unmarshal(raw, &tree); err != nil {
		return fmt.Errorf("parse cmux tree JSON: %w", err)
	}

	var pattern *regexp.Regexp
	if !allTerminals {
		p, err := regexp.Compile(match)
		if err != nil {
			return fmt.Errorf("invalid --match regex %q: %w", match, err)
		}
		pattern = p
	}

	var targets []surface
	for _, w := range tree.Windows {
		for _, ws := range w.Workspaces {
			for _, p := range ws.Panes {
				for _, s := range p.Surfaces {
					if s.Type != "terminal" {
						continue
					}
					if allTerminals || (pattern != nil && pattern.MatchString(s.Title)) {
						targets = append(targets, s)
					}
				}
			}
		}
	}

	if len(targets) == 0 {
		fmt.Fprintf(stdout, "no matching surfaces (pattern: %s)\n", match)
		return nil
	}

	for _, s := range targets {
		title := s.Title
		if len(title) > 70 {
			title = title[:70]
		}
		fmt.Fprintf(stdout, "  %-14s %s\n", s.Ref, title)
	}

	if listOnly {
		return nil
	}

	if !yes {
		fmt.Fprintf(stdout, "\nbroadcast /%s to all? [y/N] ", cmdName)
		r := bufio.NewReader(os.Stdin)
		ans, _ := r.ReadString('\n')
		ans = strings.TrimSpace(ans)
		if ans != "y" && ans != "Y" {
			fmt.Fprintln(stdout, "aborted")
			return nil
		}
	}

	text := "/" + cmdName + "\n"
	count := 0
	for _, s := range targets {
		payload, _ := json.Marshal(map[string]string{
			"surface_id": s.Ref,
			"text":       text,
		})
		if err := exec.Command("cmux", "rpc", "surface.send_text", string(payload)).Run(); err != nil {
			fmt.Fprintf(stderr, "  FAILED %s: %v\n", s.Ref, err)
			continue
		}
		count++
		fmt.Fprintf(stdout, "  sent to %s\n", s.Ref)
	}
	fmt.Fprintf(stdout, "done — broadcast to %d surfaces\n", count)
	return nil
}
