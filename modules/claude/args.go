package claude

import (
	_ "embed"
)

// ParsedArgs holds the result of parsing claude wrapper flags from raw args.
type ParsedArgs struct {
	Name            string
	Anonymous       bool
	Yolo            bool
	TaskListID      string
	ContinueSession string
	SessionName     string
	PassArgs        []string
}

// ParseArgs extracts wrapper flags (-n, -a, -t, --yolo) from args,
// passing everything else through to claude.
func ParseArgs(args []string) ParsedArgs {
	var p ParsedArgs

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				p.Name = args[i+1]
				i++
			}
		case "--anonymous", "-a":
			p.Anonymous = true
		case "--yolo":
			p.Yolo = true
		case "--continue-session", "-cs":
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				p.ContinueSession = args[i+1]
				i++
			}
		case "--session-name", "-s":
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				p.SessionName = args[i+1]
				i++
			}
		case "--task-list-id", "-t":
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				p.TaskListID = args[i+1]
				i++
			}
		default:
			p.PassArgs = append(p.PassArgs, args[i])
		}
	}

	// Fall back to session name if no task list ID
	if p.TaskListID == "" && p.Name != "" {
		p.TaskListID = p.Name
	}

	// Inject yolo args at the front of PassArgs
	if p.Yolo {
		yoloArgs := []string{"--dangerously-skip-permissions", "--append-system-prompt", yoloPrompt()}
		p.PassArgs = append(yoloArgs, p.PassArgs...)
	}

	return p
}

//go:embed prompts/safe-autonomous-mode.md
var safeAutonomousPrompt string

// yoloPrompt returns the embedded safe-autonomous-mode prompt.
func yoloPrompt() string {
	return safeAutonomousPrompt
}
