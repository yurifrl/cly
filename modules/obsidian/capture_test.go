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
			expected: []string{"-p", "/capture hello"},
		},
		{
			name:     "multiple words",
			input:    []string{"this", "is", "a", "note"},
			expected: []string{"-p", "/capture this is a note"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{"-p", "/capture "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildClaudeArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDotfilesDir(t *testing.T) {
	dir := getDotfilesDir()
	assert.Contains(t, dir, "Dotfiles")
}
