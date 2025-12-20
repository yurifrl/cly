package ai

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestContinueFlagShorthand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	Register(rootCmd)

	aiCmd, _, err := rootCmd.Find([]string{"ai"})
	assert.NoError(t, err)

	conversationID := "brave-turtle"

	aiCmd.SetArgs([]string{"-c", conversationID})
	err = aiCmd.ParseFlags([]string{"-c", conversationID})
	assert.NoError(t, err)

	value, err := aiCmd.Flags().GetString("continue")
	assert.NoError(t, err)
	assert.Equal(t, conversationID, value)
}

func TestContinueFlagLongForm(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	Register(rootCmd)

	aiCmd, _, err := rootCmd.Find([]string{"ai"})
	assert.NoError(t, err)

	conversationID := "brave-turtle"

	aiCmd.SetArgs([]string{"--continue", conversationID})
	err = aiCmd.ParseFlags([]string{"--continue", conversationID})
	assert.NoError(t, err)

	value, err := aiCmd.Flags().GetString("continue")
	assert.NoError(t, err)
	assert.Equal(t, conversationID, value)
}
