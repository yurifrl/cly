package notify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock notifier for testing
type mockNotifier struct {
	sendCalled       bool
	lastNotification Notification
	availableResult  bool
	sendError        error
}

func (m *mockNotifier) Send(ctx context.Context, n Notification) error {
	m.sendCalled = true
	m.lastNotification = n
	return m.sendError
}

func (m *mockNotifier) Available() bool {
	return m.availableResult
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

	assert.False(t, beeepMock.sendCalled)
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
