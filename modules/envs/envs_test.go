package envs

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectFieldsHonorsProfileSections(t *testing.T) {
	fields := []Field{
		{Label: "ROOT", Value: "root"},
		{Label: "WORK", Value: "work", Section: "work"},
		{Label: "PERSONAL", Value: "personal", Section: "personal"},
		{Label: "DEV", Value: "dev", Section: "dev"},
		{Label: "connection options", Value: "ignored"},
		{Label: "server", Value: "ignored"},
	}

	selected := SelectFields(fields, "work")

	require.Equal(t, []Field{
		{Label: "ROOT", Value: "root"},
		{Label: "WORK", Value: "work", Section: "work"},
		{Label: "DEV", Value: "dev", Section: "dev"},
	}, selected)
}

func TestSelectFieldsAllProfileIncludesEveryProfileSection(t *testing.T) {
	fields := []Field{
		{Label: "WORK", Value: "work", Section: "work"},
		{Label: "PERSONAL", Value: "personal", Section: "personal"},
	}

	require.Equal(t, fields, SelectFields(fields, "all"))
}
func TestOpCommandPassesOnlyTheMatchingEphemeralSession(t *testing.T) {
	command := opCommand(context.Background(), "op", sessions{
		"personal": "personal-token",
		"work":     "work-token",
	}, "personal", "item", "get", "example")

	environment := strings.Join(command.Env, "\n")
	require.Contains(t, environment, "OP_SESSION=personal-token")
	require.NotContains(t, environment, "work-token")
}

func TestLoadConfigDefaults(t *testing.T) {
	config, err := ParseConfig([]byte(`{"secrets":[{"name":"envs-ai"}]}`))

	require.NoError(t, err)
	require.Equal(t, "my.1password.com", config.DefaultAccount)
	require.Equal(t, "Private", config.DefaultVault)
	require.Equal(t, "all", config.DefaultProfile)
	require.Equal(t, "envs-ai", config.Secrets[0].Name)
}
