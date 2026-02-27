package claudesession

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yurifrl/cly/pkg/session"
)

func TestRegister(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)

	cmd, _, err := root.Find([]string{"cs"})
	require.NoError(t, err)
	assert.Equal(t, "claude-sessions", cmd.Use)
}

func TestSubcommandsRegistered(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)

	tests := []struct {
		name string
		args []string
	}{
		{"ls", []string{"cs", "ls"}},
		{"rm", []string{"cs", "rm"}},
		{"save", []string{"cs", "save"}},
		{"resume", []string{"cs", "resume"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.name, cmd.Name())
		})
	}
}

func TestRootRunENoArgs(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)

	cmd, _, err := root.Find([]string{"cs"})
	require.NoError(t, err)
	assert.NotNil(t, cmd.RunE)
}

func TestLsCmdHasAllFlag(t *testing.T) {
	cmd := lsCmd()
	flag := cmd.Flags().Lookup("all")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
	assert.Equal(t, "a", flag.Shorthand)
}

func TestSaveCmdArgs(t *testing.T) {
	cmd := saveCmd()
	assert.Equal(t, "save <name> [id]", cmd.Use)
	flag := cmd.Flags().Lookup("description")
	require.NotNil(t, flag)
	assert.Equal(t, "d", flag.Shorthand)
}

func TestResumeCmdArgs(t *testing.T) {
	cmd := resumeCmd()
	assert.Equal(t, "resume <name|id>", cmd.Use)
}

func TestExecClaudeArgs_NoYolo(t *testing.T) {
	entry := &Entry{ID: "abc-123"}
	args := execClaudeArgs(entry, false)
	assert.Equal(t, []string{"-r", "abc-123"}, args)
}

func TestExecClaudeArgs_Yolo(t *testing.T) {
	entry := &Entry{ID: "abc-123"}
	args := execClaudeArgs(entry, true)
	yolo := session.YoloArgs()
	assert.Equal(t, append(yolo, "-r", "abc-123"), args)
}
