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

// sendToStatusBar sends notification to zjstatus plugin (fire-and-forget)
func (z *ZellijNotifier) sendToStatusBar(_ context.Context, n Notification) error {
	message := fmt.Sprintf("zjstatus::notify::%s", n.Title)
	cmd := exec.Command("zellij", "pipe", message)
	return cmd.Start()
}

// sendTabUpdate sends tab emoji update to notify plugin (fire-and-forget)
func (z *ZellijNotifier) sendTabUpdate(_ context.Context) error {
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

	cmd := exec.Command("zellij", args...)
	return cmd.Start()
}
