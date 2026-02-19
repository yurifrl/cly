package claudesession

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{"save", []string{"cs", "save"}},
		{"restore", []string{"cs", "restore"}},
		{"ls", []string{"cs", "ls"}},
		{"delete", []string{"cs", "delete"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.name, cmd.Name())
		})
	}
}
