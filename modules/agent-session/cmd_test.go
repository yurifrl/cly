package agentsession

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

	cmd, _, err := root.Find([]string{"as"})
	require.NoError(t, err)
	assert.Equal(t, "agent-session", cmd.Use)
}

func TestSubcommandsRegistered(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)

	tests := []struct {
		name string
		args []string
	}{
		{"ls", []string{"as", "ls"}},
		{"rm", []string{"as", "rm"}},
		{"save", []string{"as", "save"}},
		{"resume", []string{"as", "resume"}},
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

	cmd, _, err := root.Find([]string{"as"})
	require.NoError(t, err)
	assert.NotNil(t, cmd.RunE)
}

func TestRootHasProviderFlag(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)
	cmd, _, err := root.Find([]string{"as"})
	require.NoError(t, err)

	flag := cmd.PersistentFlags().Lookup(providerFlag)
	require.NotNil(t, flag)
	assert.Equal(t, "p", flag.Shorthand)
	assert.Equal(t, defaultProvider, flag.DefValue)
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

func TestBuildResumeArgs_ClaudeNoYolo(t *testing.T) {
	provider := Provider{Name: "claude", ResumeArgs: []string{"-r", "{id}"}, YoloArgs: session.YoloArgs()}
	args := buildResumeArgs(provider, "abc-123", false)
	assert.Equal(t, []string{"-r", "abc-123"}, args)
}

func TestBuildResumeArgs_ClaudeYolo(t *testing.T) {
	provider := Provider{Name: "claude", ResumeArgs: []string{"-r", "{id}"}, YoloArgs: session.YoloArgs()}
	args := buildResumeArgs(provider, "abc-123", true)
	yolo := session.YoloArgs()
	assert.Equal(t, append(yolo, "-r", "abc-123"), args)
}

func TestBuildResumeArgs_PiYoloIgnored(t *testing.T) {
	provider := Provider{Name: "pi", ResumeArgs: []string{"--session", "{id}"}}
	args := buildResumeArgs(provider, "abc-123", true)
	assert.Equal(t, []string{"--session", "abc-123"}, args)
}
