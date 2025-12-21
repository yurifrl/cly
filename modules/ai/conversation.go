package ai

import (
	"math/rand"

	"github.com/lucasepe/codename"
)

// GenerateConversationID creates a new conversation ID with human-readable format
func GenerateConversationID() string {
	rng := rand.New(rand.NewSource(rand.Int63()))
	return codename.Generate(rng, 0)
}
