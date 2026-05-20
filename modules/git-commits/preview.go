package gitcommits

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

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

// Confirm prompts the user with single-keypress controls.
// Y / Enter = yes, n = no, r = revise (asks for guidance),
// p = prompt (asks for a free-form preprompt).
func Confirm() ConfirmResult {
	fmt.Println(style.SubtleStyle.Render("  [Y]es  [n]o  [r]evise split  [p]rompt"))
	fmt.Print("→ ")

	ch, ok := readSingleKey()
	if !ok {
		// Fall back to line-mode read (non-tty stdin, e.g. piped input)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return ConfirmResult{Action: ConfirmNo}
		}
		return classifyConfirmInput(strings.TrimSpace(input), reader)
	}

	// Echo the chosen key so the user sees what they pressed.
	switch ch {
	case '\r', '\n':
		fmt.Println("y")
	case 3, 4: // Ctrl-C / Ctrl-D
		fmt.Println()
		return ConfirmResult{Action: ConfirmNo}
	default:
		fmt.Printf("%c\n", ch)
	}

	switch ch {
	case '\r', '\n', 'y', 'Y':
		return ConfirmResult{Action: ConfirmYes}
	case 'n', 'N':
		return ConfirmResult{Action: ConfirmNo}
	case 'p', 'P':
		fmt.Print(style.BlueStyle.Render("Preprompt: "))
		reader := bufio.NewReader(os.Stdin)
		prompt, err := reader.ReadString('\n')
		if err != nil {
			return ConfirmResult{Action: ConfirmNo}
		}
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			return ConfirmResult{Action: ConfirmNo}
		}
		return ConfirmResult{Action: ConfirmRevise, Prompt: prompt}
	case 'r', 'R':
		fmt.Print(style.BlueStyle.Render("Guide the split: "))
		reader := bufio.NewReader(os.Stdin)
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
		return ConfirmResult{Action: ConfirmNo}
	}
}

// readSingleKey reads a single byte from stdin in raw mode.
// Returns ok=false when stdin is not a TTY.
func readSingleKey() (byte, bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return 0, false
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return 0, false
	}
	defer term.Restore(fd, old) //nolint:errcheck
	var buf [1]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil || n != 1 {
		return 0, false
	}
	return buf[0], true
}

// classifyConfirmInput handles the legacy line-mode path used when stdin
// is not a TTY (e.g. tests or piped input).
func classifyConfirmInput(input string, reader *bufio.Reader) ConfirmResult {
	lower := strings.ToLower(input)
	switch {
	case lower == "" || lower == "y" || lower == "yes":
		return ConfirmResult{Action: ConfirmYes}
	case lower == "n" || lower == "no":
		return ConfirmResult{Action: ConfirmNo}
	case lower == "p" || lower == "prompt":
		fmt.Print(style.BlueStyle.Render("Preprompt: "))
		prompt, err := reader.ReadString('\n')
		if err != nil {
			return ConfirmResult{Action: ConfirmNo}
		}
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			return ConfirmResult{Action: ConfirmNo}
		}
		return ConfirmResult{Action: ConfirmRevise, Prompt: prompt}
	case lower == "r" || lower == "revise":
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
