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
		{"upsert", []string{"as", "upsert"}},
		{"resume", []string{"as", "resume"}},
		{"tui", []string{"as", "tui"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.name, cmd.Name())
		})
	}
}

func TestSaveAliasWorks(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)

	cmd, _, err := root.Find([]string{"as", "save"})
	require.NoError(t, err)
	assert.Equal(t, "upsert", cmd.Name())
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
	assert.Equal(t, "all", flag.DefValue)
}

func TestLsCmdHasFlags(t *testing.T) {
	cmd := lsCmd()
	filterFlag := cmd.Flags().Lookup("filter")
	require.NotNil(t, filterFlag)
	assert.Equal(t, "f", filterFlag.Shorthand)
}

func TestParentHasScopeFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)
	cmd, _, err := root.Find([]string{"as"})
	require.NoError(t, err)

	allFlag := cmd.PersistentFlags().Lookup("all")
	require.NotNil(t, allFlag)
	assert.Equal(t, "a", allFlag.Shorthand)

	dirFlag := cmd.PersistentFlags().Lookup("directory")
	require.NotNil(t, dirFlag)
}

func TestUpsertCmdArgs(t *testing.T) {
	cmd := upsertCmd()
	assert.Equal(t, "upsert <id> [name] [description]", cmd.Use)
	assert.Contains(t, cmd.Aliases, "save")

	flag := cmd.Flags().Lookup("description")
	require.NotNil(t, flag)
	assert.Equal(t, "d", flag.Shorthand)

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag)
	assert.Equal(t, "n", nameFlag.Shorthand)

	setFlag := cmd.Flags().Lookup("set")
	require.NotNil(t, setFlag)

	metaFlag := cmd.Flags().Lookup("meta")
	require.NotNil(t, metaFlag)

	overrideFlag := cmd.Flags().Lookup("override")
	require.NotNil(t, overrideFlag)
}

func TestRmCmdHasFlags(t *testing.T) {
	cmd := rmCmd()
	filterFlag := cmd.Flags().Lookup("filter")
	require.NotNil(t, filterFlag)
	assert.Equal(t, "f", filterFlag.Shorthand)

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
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
