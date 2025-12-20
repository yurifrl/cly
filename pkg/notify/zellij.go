package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ZellijNotifier sends notifications to Zellij status bar and tabs
type ZellijNotifier struct {
	eventType   string // "notification", "stop", "posttooluse", etc.
	useStatus   bool   // Send to zjstatus plugin
	useNotify   bool   // Send to notify plugin
}

// NewZellijNotifier creates a new Zellij notifier with the given event type
func NewZellijNotifier(eventType string, useStatus, useNotify bool) *ZellijNotifier {
	return &ZellijNotifier{
		eventType: eventType,
		useStatus: useStatus,
		useNotify: useNotify,
	}
}

// Send sends notification to Zellij
func (z *ZellijNotifier) Send(ctx context.Context, n Notification) error {
	if !z.Available() {
		return nil
	}

	// Send to status bar (zjstatus plugin) if enabled
	if z.useStatus {
		if err := z.sendToStatusBar(ctx, n); err != nil {
			// Continue even if status bar fails
		}
	}

	// Send tab update (notify plugin) if enabled
	if z.useNotify {
		if err := z.sendTabUpdate(ctx); err != nil {
			// Continue even if tab update fails
		}
	}

	return nil
}

// Available returns true if we're in a Zellij session
func (z *ZellijNotifier) Available() bool {
	return os.Getenv("ZELLIJ") != ""
}

// sendToStatusBar sends notification to zjstatus plugin
func (z *ZellijNotifier) sendToStatusBar(ctx context.Context, n Notification) error {
	// Use title for status bar message
	message := fmt.Sprintf("zjstatus::notify::%s", n.Title)

	cmd := exec.CommandContext(ctx, "zellij", "pipe", message)
	return cmd.Run()
}

// sendTabUpdate sends tab emoji update to notify plugin
func (z *ZellijNotifier) sendTabUpdate(ctx context.Context) error {
	paneID := os.Getenv("ZELLIJ_PANE_ID")
	sessionName := os.Getenv("ZELLIJ_SESSION_NAME")

	args := []string{"pipe", "-n", "notify"}

	if paneID != "" {
		args = append(args, "-a", fmt.Sprintf("pane_id=%s", paneID))
	}

	if sessionName != "" {
		args = append(args, "-a", fmt.Sprintf("session_name=%s", sessionName))
	}

	args = append(args, z.eventType)

	cmd := exec.CommandContext(ctx, "zellij", args...)
	return cmd.Run()
}
