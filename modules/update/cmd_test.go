package update

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	Register(rootCmd)

	// Find the update command
	var updateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			updateCmd = cmd
			break
		}
	}

	require.NotNil(t, updateCmd, "update command should be registered")
	assert.Equal(t, "update", updateCmd.Use)
	assert.NotEmpty(t, updateCmd.Short)
}

func TestCommandFlags(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	Register(rootCmd)

	// Find the update command
	var updateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			updateCmd = cmd
			break
		}
	}

	require.NotNil(t, updateCmd)

	// Check flags exist
	checkFlag := updateCmd.Flags().Lookup("check")
	assert.NotNil(t, checkFlag, "--check flag should exist")

	forceFlag := updateCmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag, "--force flag should exist")

	yesFlag := updateCmd.Flags().Lookup("yes")
	assert.NotNil(t, yesFlag, "--yes flag should exist")

	versionFlag := updateCmd.Flags().Lookup("version")
	assert.NotNil(t, versionFlag, "--version flag should exist")

	// Check short flag for yes
	yesFlagShort := updateCmd.Flags().ShorthandLookup("y")
	assert.NotNil(t, yesFlagShort, "-y flag should exist")
}
