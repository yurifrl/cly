package ai

import (
	"fmt"
	"time"
)

// GenerateConversationID creates a new conversation ID with timestamp
func GenerateConversationID() string {
	return fmt.Sprintf("modsi-%d", time.Now().Unix())
}
