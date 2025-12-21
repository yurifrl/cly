package notify

import "context"

// Notifier is the interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, n Notification) error
	Available() bool
}

// MultiNotifier sends notifications to multiple independent notifiers
type MultiNotifier struct {
	notifiers []Notifier // All enabled notifiers
}

// Send sends notification to all available notifiers (independently, not fallback)
func (m *MultiNotifier) Send(ctx context.Context, n Notification) error {
	for _, notifier := range m.notifiers {
		if notifier != nil && notifier.Available() {
			if err := notifier.Send(ctx, n); err != nil {
				// Log error but continue to other notifiers
				// We don't want to fail if one notifier fails
			}
		}
	}
	return nil
}

// Available returns true if at least one notifier is available
func (m *MultiNotifier) Available() bool {
	for _, notifier := range m.notifiers {
		if notifier != nil && notifier.Available() {
			return true
		}
	}
	return false
}

// NewMultiNotifier creates a MultiNotifier with specified notifiers
func NewMultiNotifier(notifiers []Notifier) Notifier {
	return &MultiNotifier{
		notifiers: notifiers,
	}
}

// New creates a new MultiNotifier with all enabled notifiers based on config
func New(eventType string, useZellijStatus, useZellijNotify bool, icon string) Notifier {
	var notifiers []Notifier

	// Always use beeep for macOS notifications
	notifiers = append(notifiers, &BeeepNotifier{Icon: icon})

	if useZellijStatus || useZellijNotify {
		notifiers = append(notifiers, NewZellijNotifier(eventType, useZellijStatus, useZellijNotify))
	}

	return NewMultiNotifier(notifiers)
}
