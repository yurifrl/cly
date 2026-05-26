package notify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock notifier for testing
type mockNotifier struct {
	sendCalled       bool
	lastNotification Notification
	availableResult  bool
	sendError        error
	events           chan ActionEvent
}

func (m *mockNotifier) Send(ctx context.Context, n Notification) error {
	m.sendCalled = true
	m.lastNotification = n
	return m.sendError
}

func (m *mockNotifier) Available() bool {
	return m.availableResult
}

func (m *mockNotifier) Events() <-chan ActionEvent {
	if m.events == nil {
		return closedActionChan()
	}
	return m.events
}

func TestMultiNotifier_SendsToAllNotifiers(t *testing.T) {
	beeepMock := &mockNotifier{availableResult: true}
	terminalMock := &mockNotifier{availableResult: true}
	zellijMock := &mockNotifier{availableResult: true}

	multi := NewMultiNotifier([]Notifier{beeepMock, terminalMock, zellijMock})

	n := Notification{
		Title:   "Test",
		Message: "Test message",
	}

	err := multi.Send(context.Background(), n)
	require.NoError(t, err)

	assert.True(t, beeepMock.sendCalled)
	assert.True(t, terminalMock.sendCalled)
	assert.True(t, zellijMock.sendCalled)
	assert.Equal(t, "Test", beeepMock.lastNotification.Title)
}

func TestMultiNotifier_SkipsUnavailableNotifiers(t *testing.T) {
	beeepMock := &mockNotifier{availableResult: false}
	terminalMock := &mockNotifier{availableResult: true}
	zellijMock := &mockNotifier{availableResult: true}

	multi := NewMultiNotifier([]Notifier{beeepMock, terminalMock, zellijMock})

	n := Notification{Title: "Test"}

	err := multi.Send(context.Background(), n)
	require.NoError(t, err)

	assert.False(t, beeepMock.sendCalled, "Should skip unavailable notifier")
	assert.True(t, terminalMock.sendCalled)
	assert.True(t, zellijMock.sendCalled)
}

func TestMultiNotifier_Available(t *testing.T) {
	tests := []struct {
		name      string
		available []bool
		expected  bool
	}{
		{"all available", []bool{true, true, true}, true},
		{"some available", []bool{true, false, true}, true},
		{"none available", []bool{false, false, false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notifiers []Notifier
			for _, avail := range tt.available {
				notifiers = append(notifiers, &mockNotifier{availableResult: avail})
			}
			multi := NewMultiNotifier(notifiers)

			assert.Equal(t, tt.expected, multi.Available())
		})
	}
}

func TestMultiNotifier_EventsFanIn(t *testing.T) {
	a := &mockNotifier{availableResult: true, events: make(chan ActionEvent, 4)}
	b := &mockNotifier{availableResult: true, events: make(chan ActionEvent, 4)}
	multi := NewMultiNotifier([]Notifier{a, b})

	a.events <- ActionEvent{Group: "ga", ActionID: "snooze"}
	b.events <- ActionEvent{Group: "gb", ActionID: "retry"}

	got := map[string]string{}
	for i := 0; i < 2; i++ {
		select {
		case ev := <-multi.Events():
			got[ev.Group] = ev.ActionID
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for fan-in event %d", i)
		}
	}
	assert.Equal(t, "snooze", got["ga"])
	assert.Equal(t, "retry", got["gb"])

	close(a.events)
	close(b.events)
}

func TestMultiNotifier_EventsNilSafe(t *testing.T) {
	// Backends without action callbacks return closedActionChan().
	a := &mockNotifier{availableResult: true}
	b := &mockNotifier{availableResult: true}
	multi := NewMultiNotifier([]Notifier{a, b})

	select {
	case <-multi.Events():
		// fine — nothing expected
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestNotification_ActionsRoundTrip(t *testing.T) {
	n := Notification{
		Group: "cly.every.foo",
		Actions: []Action{
			{ID: "snooze", Title: "Snooze 5m"},
			{ID: "dismiss", Title: "Dismiss"},
		},
	}
	require.Len(t, n.Actions, 2)
	assert.Equal(t, "snooze", n.Actions[0].ID)
	assert.Equal(t, "Dismiss", n.Actions[1].Title)
}
