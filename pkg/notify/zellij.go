package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ZellijNotifier sends notifications to Zellij status bar and tabs
type ZellijNotifier struct {
	eventType    string // "notification", "stop", "posttooluse", etc.
	useStatus    bool   // Send to zjstatus plugin
	useNotify    bool   // Send to old notify plugin
	useAttention bool   // Send to zellij-attention plugin
}

// NewZellijNotifier creates a new Zellij notifier with the given event type
func NewZellijNotifier(eventType string, useStatus, useNotify, useAttention bool) *ZellijNotifier {
	return &ZellijNotifier{
		eventType:    eventType,
		useStatus:    useStatus,
		useNotify:    useNotify,
		useAttention: useAttention,
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

	// Send tab update (old notify plugin) if enabled
	if z.useNotify {
		if err := z.sendNotifyTabUpdate(ctx); err != nil {
			// Continue even if tab update fails
		}
	}

	// Send tab update (zellij-attention plugin) if enabled
	if z.useAttention {
		if err := z.sendAttentionTabUpdate(ctx); err != nil {
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

// sendNotifyTabUpdate sends tab emoji update to old notify plugin (fire-and-forget)
func (z *ZellijNotifier) sendNotifyTabUpdate(_ context.Context) error {
	paneID := os.Getenv("ZELLIJ_PANE_ID")
	sessionName := os.Getenv("ZELLIJ_SESSION_NAME")
	args := buildNotifyArgs(z.eventType, paneID, sessionName)
	cmd := exec.Command("zellij", args...)
	return cmd.Start()
}

// buildNotifyArgs builds args for the old notify plugin pipe command
func buildNotifyArgs(eventType, paneID, sessionName string) []string {
	args := []string{"pipe", "-n", "notify"}
	if paneID != "" {
		args = append(args, "-a", fmt.Sprintf("pane_id=%s", paneID))
	}
	if sessionName != "" {
		args = append(args, "-a", fmt.Sprintf("session_name=%s", sessionName))
	}
	args = append(args, eventType)
	return args
}

// sendAttentionTabUpdate sends tab update to zellij-attention plugin (fire-and-forget)
func (z *ZellijNotifier) sendAttentionTabUpdate(_ context.Context) error {
	paneID := os.Getenv("ZELLIJ_PANE_ID")
	pipeName := buildAttentionPipeName(z.eventType, paneID)
	cmd := exec.Command("zellij", "pipe", "--name", pipeName)
	return cmd.Start()
}

// buildAttentionPipeName builds the pipe name for zellij-attention plugin
func buildAttentionPipeName(eventType, paneID string) string {
	state := mapEventToAttentionState(eventType)
	return fmt.Sprintf("zellij-attention::%s::%s", state, paneID)
}

// mapEventToAttentionState maps hook event types to zellij-attention states
func mapEventToAttentionState(eventType string) string {
	switch eventType {
	case "notification":
		return "waiting"
	case "stop":
		return "completed"
	default:
		return eventType
	}
}
