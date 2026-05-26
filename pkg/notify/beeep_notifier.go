package notify

import (
	"context"

	"github.com/gen2brain/beeep"
)

// BeeepNotifier sends notifications using the beeep library
type BeeepNotifier struct {
	Icon string // Path to icon file
}

// Send sends a notification using beeep.Alert
func (b *BeeepNotifier) Send(ctx context.Context, n Notification) error {
	return beeep.Notify(n.Title, n.Message, b.Icon)
}

// Available returns true (beeep handles platform detection internally)
func (b *BeeepNotifier) Available() bool {
	return true
}

// Events returns a closed channel; beeep does not support action callbacks.
func (b *BeeepNotifier) Events() <-chan ActionEvent {
	return closedActionChan()
}
