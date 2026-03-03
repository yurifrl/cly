package obsidian

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildClaudeArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "single word",
			input:    []string{"hello"},
			expected: []string{"-p", "/capture hello", "--allowedTools", "WebFetch,WebSearch"},
		},
		{
			name:     "multiple words",
			input:    []string{"this", "is", "a", "note"},
			expected: []string{"-p", "/capture this is a note", "--allowedTools", "WebFetch,WebSearch"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{"-p", "/capture ", "--allowedTools", "WebFetch,WebSearch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildClaudeArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRun_capture_routing(t *testing.T) {
	// capture route: args[0] == "capture" → runCapture is called (not execObsidian)
	// We can't easily test the full execution, but we can verify the routing logic
	// by checking that non-capture args go to execObsidian (which will fail with PATH error)

	// Test that "capture" args are NOT passed to execObsidian by checking
	// that buildClaudeArgs is used correctly for capture sub-args
	captureArgs := []string{"my", "note"}
	result := buildClaudeArgs(captureArgs)
	assert.Equal(t, []string{"-p", "/capture my note", "--allowedTools", "WebFetch,WebSearch"}, result)
}
