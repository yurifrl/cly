package notify

import "context"

// Notifier is the interface for sending notifications.
type Notifier interface {
	Send(ctx context.Context, n Notification) error
	Available() bool
}

// MultiNotifier sends notifications to multiple independent notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

// Send sends notification to all available notifiers.
func (m *MultiNotifier) Send(ctx context.Context, n Notification) error {
	for _, notifier := range m.notifiers {
		if notifier != nil && notifier.Available() {
			_ = notifier.Send(ctx, n)
		}
	}
	return nil
}

// Available returns true if at least one notifier is available.
func (m *MultiNotifier) Available() bool {
	for _, notifier := range m.notifiers {
		if notifier != nil && notifier.Available() {
			return true
		}
	}
	return false
}

// NewMultiNotifier creates a MultiNotifier.
func NewMultiNotifier(notifiers []Notifier) Notifier {
	return &MultiNotifier{notifiers: notifiers}
}

// New creates a new MultiNotifier with all enabled notifiers based on config.
// On darwin the native UNUserNotificationCenter backend is preferred; on
// failure (placeholder bundle, codesign mismatch) it falls back to beeep.
func New(eventType string, useZellijStatus, useZellijNotify, useZellijAttention bool, icon string) Notifier {
	var notifiers []Notifier

	native := NewNativeMacOSNotifier(context.Background())
	if native.Available() {
		notifiers = append(notifiers, native)
	} else {
		notifiers = append(notifiers, &BeeepNotifier{Icon: icon})
	}

	if useZellijStatus || useZellijNotify || useZellijAttention {
		notifiers = append(notifiers, NewZellijNotifier(eventType, useZellijStatus, useZellijNotify, useZellijAttention))
	}

	return NewMultiNotifier(notifiers)
}
