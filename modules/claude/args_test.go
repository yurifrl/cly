package claude

import (
	"github.com/stretchr/testify/require"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yurifrl/cly/pkg/session"
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
				PassArgs: []string{"--dangerously-skip-permissions", "--append-system-prompt", session.YoloArgs()[2], "-p", "do stuff"},
			},
		},
		{
			name: "yolo with name",
			args: []string{"-n", "proj", "--yolo"},
			expected: ParsedArgs{
				Name:       "proj",
				TaskListID: "proj",
				Yolo:       true,
				PassArgs:   []string{"--dangerously-skip-permissions", "--append-system-prompt", session.YoloArgs()[2]},
			},
		},
		{
			name:     "cli is a wrapper flag",
			args:     []string{"--cli", "--yolo"},
			expected: ParsedArgs{PlainCLI: true, Yolo: true, PassArgs: session.YoloArgs()},
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: ParsedArgs{},
		},
		{
			name:     "session name long",
			args:     []string{"--session-name", "myproj", "-p", "hello"},
			expected: ParsedArgs{SessionName: "myproj", PassArgs: []string{"-p", "hello"}},
		},
		{
			name:     "session name short",
			args:     []string{"-s", "myproj"},
			expected: ParsedArgs{SessionName: "myproj"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseArgs(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseArgs_WrapperFlags(t *testing.T) {
	p := ParseArgs([]string{"--no-wrappers", "--no-headroom", "-p", "hi"})
	assert.True(t, p.NoWrappers)
	assert.True(t, p.NoHeadroom)
	assert.Equal(t, []string{"-p", "hi"}, p.PassArgs)
}

func TestHeadroomEnabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	assert.True(t, headroomEnabled(ParsedArgs{}))
	assert.False(t, headroomEnabled(ParsedArgs{NoHeadroom: true}))
	assert.False(t, headroomEnabled(ParsedArgs{NoWrappers: true}))
}

func TestHeadroomProxyURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Setenv("CLY_MODULES_CLAUDE_HEADROOM_URL", "http://"+ln.Addr().String())
	assert.Equal(t, "http://"+ln.Addr().String(), headroomProxyURL(ParsedArgs{}))
	assert.Equal(t, "", headroomProxyURL(ParsedArgs{NoHeadroom: true}))
	ln.Close()
	assert.Equal(t, "", headroomProxyURL(ParsedArgs{}))
}
