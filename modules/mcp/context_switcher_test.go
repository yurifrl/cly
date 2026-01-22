package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextOptionsOrder(t *testing.T) {
	// claude:project should be first (index 0) so it starts selected
	first := contextOptions[0]
	assert.Equal(t, "claude", first.ai)
	assert.Equal(t, "project", first.scope)
	assert.Equal(t, "Claude Code (project)", first.label)
}

func TestFormatMCPCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{"zero MCPs", 0, "(empty)"},
		{"one MCP", 1, "[1 MCP]"},
		{"multiple MCPs", 3, "[3 MCPs]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMCPCount(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}
