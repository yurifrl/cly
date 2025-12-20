package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotification_Struct(t *testing.T) {
	n := Notification{
		Title:    "Test Title",
		Subtitle: "Test Subtitle",
		Message:  "Test Message",
		Sound:    "Glass",
		Group:    "test-group",
	}

	assert.Equal(t, "Test Title", n.Title)
	assert.Equal(t, "Test Subtitle", n.Subtitle)
	assert.Equal(t, "Test Message", n.Message)
	assert.Equal(t, "Glass", n.Sound)
	assert.Equal(t, "test-group", n.Group)
}
