package notify

import (
	"context"
	"os/exec"
)

// TerminalNotifier sends notifications using terminal-notifier on macOS
type TerminalNotifier struct {
	Icon string // Path to icon file
}

// Send sends a notification using terminal-notifier
func (t *TerminalNotifier) Send(ctx context.Context, n Notification) error {
	// Debug what we received
	println("DEBUG terminal-notifier RECEIVED:")
	println("  Title:", n.Title)
	println("  Subtitle:", n.Subtitle)
	println("  Message:", n.Message)

	// Combine subtitle with message context for visibility
	subtitle := n.Subtitle
	if subtitle != "" && n.Message != "" {
		subtitle = subtitle + " - " + n.Message
	} else if n.Message != "" {
		subtitle = n.Message
	}

	args := []string{
		"-title", n.Title,
	}

	if subtitle != "" {
		args = append(args, "-subtitle", subtitle)
	}

	println("DEBUG terminal-notifier SENDING:")
	println("  Title:", n.Title)
	println("  Subtitle:", subtitle)

	if n.Sound != "" {
		args = append(args, "-sound", n.Sound)
	}

	if n.Group != "" {
		args = append(args, "-group", n.Group)
	}

	if t.Icon != "" {
		args = append(args, "-appIcon", t.Icon)
	}

	cmd := exec.CommandContext(ctx, "terminal-notifier", args...)

	println("DEBUG terminal-notifier COMMAND:")
	print("   terminal-notifier")
	for _, arg := range args {
		print(" ", arg)
	}
	println()

	return cmd.Run()
}

// Available returns true if terminal-notifier is installed
func (t *TerminalNotifier) Available() bool {
	_, err := exec.LookPath("terminal-notifier")
	return err == nil
}
