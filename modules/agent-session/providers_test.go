package agentsession

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderByName_Defaults(t *testing.T) {
	provider, err := providerByName("claude")
	require.NoError(t, err)
	assert.Equal(t, "claude", provider.Name)
	assert.Equal(t, "claude", provider.Command)
	assert.Equal(t, []string{"-r", "{id}"}, provider.ResumeArgs)

	provider, err = providerByName("pi")
	require.NoError(t, err)
	assert.Equal(t, "pi", provider.Name)
	assert.Equal(t, "pi", provider.Command)
	assert.Equal(t, []string{"--session", "{id}"}, provider.ResumeArgs)
}

func TestProviderByName_Unknown(t *testing.T) {
	_, err := providerByName("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestAvailableProviders(t *testing.T) {
	providers := availableProviders()
	assert.Contains(t, providers, "claude")
	assert.Contains(t, providers, "pi")
}
