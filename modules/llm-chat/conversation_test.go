package llmchat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateConversationID(t *testing.T) {
	id := GenerateConversationID()

	assert.NotEmpty(t, id, "ID should not be empty")
	assert.True(t, strings.Contains(id, "-"), "ID should be hyphenated (adjective-noun format)")
	assert.Greater(t, len(id), 5, "ID should be at least a few characters")
}
