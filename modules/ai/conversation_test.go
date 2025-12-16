package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateConversationID(t *testing.T) {
	id := GenerateConversationID()

	assert.True(t, strings.HasPrefix(id, "modsi-"), "ID should start with modsi-")
	assert.Greater(t, len(id), len("modsi-"), "ID should have timestamp")
}
