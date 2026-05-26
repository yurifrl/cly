package notify

import "context"

// Notifier is the interface for sending notifications.
//
// Events returns a channel that emits ActionEvent values whenever the user
// clicks an action button on a notification. Backends that don't support
// action callbacks (beeep, zellij) return a closed channel — receiving from
// it never blocks and never delivers a value, so callers can range over it
// safely without a nil-check.
type Notifier interface {
	Send(ctx context.Context, n Notification) error
	Available() bool
	Events() <-chan ActionEvent
}

// closedActionChan returns a pre-closed channel for backends that never emit
// action events. Receiving from it never blocks.
func closedActionChan() <-chan ActionEvent {
	ch := make(chan ActionEvent)
	close(ch)
	return ch
}

// MultiNotifier sends notifications to multiple independent notifiers and
// fans in their ActionEvent streams into a single channel.
type MultiNotifier struct {
	notifiers []Notifier
	events    chan ActionEvent
}

// Send sends notification to all available notifiers (independently, not fallback)
func (m *MultiNotifier) Send(ctx context.Context, n Notification) error {
	for _, notifier := range m.notifiers {
		if notifier != nil && notifier.Available() {
			if err := notifier.Send(ctx, n); err != nil {
				// Log error but continue to other notifiers
				// We don't want to fail if one notifier fails
				_ = err
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

// Events returns a fan-in channel of all child notifiers' events.
func (m *MultiNotifier) Events() <-chan ActionEvent {
	return m.events
}

// NewMultiNotifier creates a MultiNotifier with specified notifiers.
// It spawns a goroutine per child to fan in ActionEvents.
func NewMultiNotifier(notifiers []Notifier) Notifier {
	m := &MultiNotifier{
		notifiers: notifiers,
		events:    make(chan ActionEvent, 16),
	}
	for _, n := range notifiers {
		if n == nil {
			continue
		}
		ch := n.Events()
		if ch == nil {
			continue
		}
		go func(in <-chan ActionEvent) {
			for ev := range in {
				m.events <- ev
			}
		}(ch)
	}
	return m
}

// New creates a new MultiNotifier with all enabled notifiers based on config.
// On darwin, the native UNUserNotificationCenter backend is preferred when
// the embedded bundle is present and codesign-verified; otherwise it falls
// back to beeep with a one-line stderr warning.
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
