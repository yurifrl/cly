package zs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSessionNames(t *testing.T) {
	output := "work\ndev\ntest (current)\n"
	assert.Equal(t, []string{"work", "dev", "test"}, parseSessionNames(output))
}

func TestSessionExists(t *testing.T) {
	assert.True(t, SessionExists("work", []string{"work", "dev"}))
	assert.False(t, SessionExists("nope", []string{"work", "dev"}))
}

func TestSessionNameForDir(t *testing.T) {
	assert.Equal(t, "my_project", SessionNameForDir("/tmp/my.project", ""))
	assert.Equal(t, "custom:name", SessionNameForDir("/tmp/my.project", "custom:name"))
}

func TestBuildCreateSessionArgs(t *testing.T) {
	assert.Equal(t,
		[]string{"zellij", "--session", "alpha", "--new-session-with-layout", "default", "options", "--default-cwd", "/repo/alpha"},
		BuildCreateSessionArgs("alpha", "/repo/alpha", "default"),
	)

	assert.Equal(t,
		[]string{"zellij", "--session", "alpha", "--new-session-with-layout", "/layouts/dev.kdl", "options", "--default-cwd", "/repo/alpha"},
		BuildCreateSessionArgs("alpha", "/repo/alpha", "/layouts/dev.kdl"),
	)

	assert.Equal(t,
		[]string{"zellij", "--session", "alpha", "--new-session-with-layout", "default", "options", "--default-cwd", "/repo/alpha"},
		BuildCreateSessionArgs("alpha", "/repo/alpha", ""),
	)
}

func TestBuildNewTabArgs(t *testing.T) {
	assert.Equal(t,
		[]string{"action", "new-tab", "--layout", "default", "--name", "alpha", "--cwd", "/repo/alpha"},
		BuildNewTabArgs("alpha", "/repo/alpha", "default"),
	)
}
