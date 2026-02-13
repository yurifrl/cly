package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected ParsedArgs
	}{
		{
			name:     "no flags",
			args:     []string{"-p", "hello"},
			expected: ParsedArgs{PassArgs: []string{"-p", "hello"}},
		},
		{
			name:     "name flag long falls back to task list id",
			args:     []string{"--name", "myproject", "-p", "hello"},
			expected: ParsedArgs{Name: "myproject", TaskListID: "myproject", PassArgs: []string{"-p", "hello"}},
		},
		{
			name:     "name flag short falls back to task list id",
			args:     []string{"-n", "myproject"},
			expected: ParsedArgs{Name: "myproject", TaskListID: "myproject"},
		},
		{
			name:     "anonymous flag",
			args:     []string{"-a", "--resume"},
			expected: ParsedArgs{Anonymous: true, PassArgs: []string{"--resume"}},
		},
		{
			name:     "task list id long",
			args:     []string{"--task-list-id", "abc123", "-p", "do stuff"},
			expected: ParsedArgs{TaskListID: "abc123", PassArgs: []string{"-p", "do stuff"}},
		},
		{
			name:     "task list id short",
			args:     []string{"-t", "abc123"},
			expected: ParsedArgs{TaskListID: "abc123"},
		},
		{
			name: "all flags combined",
			args: []string{"-n", "proj", "-t", "task1", "-p", "hello"},
			expected: ParsedArgs{
				Name:       "proj",
				TaskListID: "task1",
				PassArgs:   []string{"-p", "hello"},
			},
		},
		{
			name:     "task list id not overridden by name fallback",
			args:     []string{"-n", "proj", "-t", "explicit-task"},
			expected: ParsedArgs{Name: "proj", TaskListID: "explicit-task"},
		},
		{
			name:     "continue session long",
			args:     []string{"--continue-session", "myproj", "-p", "hello"},
			expected: ParsedArgs{ContinueSession: "myproj", PassArgs: []string{"-p", "hello"}},
		},
		{
			name:     "continue session short",
			args:     []string{"-cs", "myproj"},
			expected: ParsedArgs{ContinueSession: "myproj"},
		},
		{
			name: "yolo flag injects skip-permissions and system prompt",
			args: []string{"--yolo", "-p", "do stuff"},
			expected: ParsedArgs{
				Yolo:     true,
				PassArgs: []string{"--dangerously-skip-permissions", "--append-system-prompt", yoloPrompt(), "-p", "do stuff"},
			},
		},
		{
			name: "yolo with name",
			args: []string{"-n", "proj", "--yolo"},
			expected: ParsedArgs{
				Name:       "proj",
				TaskListID: "proj",
				Yolo:       true,
				PassArgs:   []string{"--dangerously-skip-permissions", "--append-system-prompt", yoloPrompt()},
			},
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: ParsedArgs{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseArgs(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}
