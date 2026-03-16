package llmchat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConversationID_Generated(t *testing.T) {
	id := GenerateConversationID()
	assert.NotEmpty(t, id)
}

func TestConversationID_Unique(t *testing.T) {
	id1 := GenerateConversationID()
	id2 := GenerateConversationID()
	// Very unlikely to be equal
	assert.NotEqual(t, id1, id2)
}
