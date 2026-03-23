package gitcommits

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yurifrl/cly/pkg/style"
)

// RenderPlan renders a human-readable plan preview.
func RenderPlan(plan *CommitPlan) string {
	var b strings.Builder

	b.WriteString(style.TitleStyle.Render(fmt.Sprintf("Split plan — %d commits:", len(plan.Groups))))
	b.WriteString("\n\n")

	for i, g := range plan.Groups {
		// Commit number and title
		b.WriteString(style.GreenStyle.Render(fmt.Sprintf("%d. %s", i+1, g.Title)))
		b.WriteString("\n")

		// Summary
		if g.Summary != "" {
			b.WriteString(fmt.Sprintf("   %s\n", g.Summary))
		}

		// Files
		for _, f := range g.Files {
			icon := statusIcon(f.Status)
			path := f.Path
			if f.OldPath != "" {
				path = fmt.Sprintf("%s → %s", f.OldPath, f.Path)
			}
			if len(f.HunkIDs) > 0 && !f.WholeFile {
				b.WriteString(fmt.Sprintf("   %s %s [%s]\n", icon, path, strings.Join(f.HunkIDs, ",")))
			} else {
				b.WriteString(fmt.Sprintf("   %s %s\n", icon, path))
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}

// RenderJSON outputs the plan as JSON to stdout.
func RenderJSON(plan *CommitPlan) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

// ConfirmAction represents the user's choice after previewing a plan.
type ConfirmAction int

const (
	ConfirmYes    ConfirmAction = iota // Execute the plan
	ConfirmNo                          // Abort
	ConfirmRevise                      // Re-plan with a guidance prompt
)

// ConfirmResult holds the user's choice and optional revision prompt.
type ConfirmResult struct {
	Action ConfirmAction
	Prompt string // Non-empty when Action == ConfirmRevise
}

// Confirm prompts the user for confirmation with 3 options.
// Returns the action chosen and an optional re-plan prompt.
func Confirm() ConfirmResult {
	fmt.Println(style.SubtleStyle.Render("  [Y]es  [n]o  [r]evise split"))
	fmt.Print("→ ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ConfirmResult{Action: ConfirmNo}
	}
	input = strings.TrimSpace(input)

	lower := strings.ToLower(input)
	switch {
	case lower == "" || lower == "y" || lower == "yes":
		return ConfirmResult{Action: ConfirmYes}
	case lower == "n" || lower == "no":
		return ConfirmResult{Action: ConfirmNo}
	case lower == "r" || lower == "revise":
		// Ask for the guidance prompt
		fmt.Print(style.BlueStyle.Render("Guide the split: "))
		prompt, err := reader.ReadString('\n')
		if err != nil {
			return ConfirmResult{Action: ConfirmNo}
		}
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			return ConfirmResult{Action: ConfirmNo}
		}
		return ConfirmResult{Action: ConfirmRevise, Prompt: prompt}
	default:
		// Treat anything else as a revision prompt directly
		// e.g. user typed "combine commits 1 and 3" without pressing 'r' first
		return ConfirmResult{Action: ConfirmRevise, Prompt: input}
	}
}

func statusIcon(status FileStatus) string {
	switch status {
	case StatusAdded:
		return style.GreenStyle.Render("A")
	case StatusModified:
		return style.BlueStyle.Render("M")
	case StatusDeleted:
		return style.RedStyle.Render("D")
	case StatusRenamed:
		return style.YellowStyle.Render("R")
	default:
		return "?"
	}
}
