package notify

import (
	"context"

	"github.com/gen2brain/beeep"
)

// BeeepNotifier sends notifications using the beeep library
type BeeepNotifier struct {
	Icon string // Path to icon file
}

// Send sends a notification using beeep
func (b *BeeepNotifier) Send(ctx context.Context, n Notification) error {
	// Combine subtitle with message context for visibility in notification center
	subtitle := n.Subtitle
	if subtitle != "" && n.Message != "" {
		subtitle = subtitle + " - " + n.Message
	} else if n.Message != "" {
		subtitle = n.Message
	}

	title := b.combineTitle(n.Title, subtitle)

	// Debug: print what we're sending
	println("DEBUG: Sending beeep notification")
	println("  Title:", title)
	println("  Message: (empty)")
	println("  Icon:", b.Icon)

	return beeep.Notify(title, "", b.Icon)
}

// Available returns true (beeep handles platform detection internally)
func (b *BeeepNotifier) Available() bool {
	return true
}

// combineTitle combines title and subtitle for display
func (b *BeeepNotifier) combineTitle(title, subtitle string) string {
	if subtitle != "" {
		return title + " - " + subtitle
	}
	return title
}
