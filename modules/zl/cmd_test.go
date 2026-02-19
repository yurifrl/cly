package zl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSwitchInside_SessionExists(t *testing.T) {
	// Test that when inside zellij and session exists, we switch to it
	t.Skip("Integration test - requires actual zellij session")
}

func TestSwitchInside_SessionDoesNotExist(t *testing.T) {
	// Test that when inside zellij and session doesn't exist, we create then switch
	t.Skip("Integration test - requires actual zellij session")
}

func TestListSessions(t *testing.T) {
	t.Run("parses session list output", func(t *testing.T) {
		output := `work
dev
test (current)`
		sessions := ListSessions(output)
		assert.Contains(t, sessions, "work")
		assert.Contains(t, sessions, "dev")
		assert.Contains(t, sessions, "test")
		assert.Len(t, sessions, 3)
	})

	t.Run("handles empty output", func(t *testing.T) {
		sessions := ListSessions("")
		assert.Empty(t, sessions)
	})
}

func TestSessionExists(t *testing.T) {
	t.Run("returns true when session in list", func(t *testing.T) {
		sessions := []string{"work", "dev", "test"}
		assert.True(t, SessionExists("work", sessions))
	})

	t.Run("returns false when session not in list", func(t *testing.T) {
		sessions := []string{"work", "dev", "test"}
		assert.False(t, SessionExists("nonexistent", sessions))
	})

	t.Run("returns false for empty list", func(t *testing.T) {
		assert.False(t, SessionExists("work", []string{}))
	})
}
