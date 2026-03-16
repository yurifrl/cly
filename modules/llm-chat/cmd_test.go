package llmchat

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	Register(rootCmd)

	aiCmd, _, err := rootCmd.Find([]string{"llm-chat"})
	assert.NoError(t, err)
	assert.Equal(t, "llm-chat", aiCmd.Use)
}

func TestApiFlagDefault(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	Register(rootCmd)

	aiCmd, _, _ := rootCmd.Find([]string{"llm-chat"})
	value, err := aiCmd.Flags().GetString("api")
	assert.NoError(t, err)
	assert.Equal(t, "anthropic", value)
}
