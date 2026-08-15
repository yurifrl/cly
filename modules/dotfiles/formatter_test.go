package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSSHKey(t *testing.T) {
	dir := t.TempDir()
	privateKey := filepath.Join(dir, "github-signing")

	originalRun := sshKeygenPublicRun
	t.Cleanup(func() { sshKeygenPublicRun = originalRun })
	sshKeygenPublicRun = func(path string) ([]byte, error) {
		assert.Equal(t, privateKey, path)
		return []byte("ssh-ed25519 derived-public-key\n"), nil
	}

	require.NoError(t, os.WriteFile(privateKey, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----\n"), 0644))
	require.NoError(t, formatSSHKey(privateKey))

	privateInfo, err := os.Stat(privateKey)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), privateInfo.Mode().Perm())
	publicKey, err := os.ReadFile(privateKey + ".pub")
	require.NoError(t, err)
	assert.Equal(t, "ssh-ed25519 derived-public-key\n", string(publicKey))
	publicInfo, err := os.Stat(privateKey + ".pub")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), publicInfo.Mode().Perm())
}

func TestFormatSSHKeyRejectsNonKey(t *testing.T) {
	privateKey := filepath.Join(t.TempDir(), "not-a-key")
	require.NoError(t, os.WriteFile(privateKey, []byte("not a private key\n"), 0600))

	err := formatSSHKey(privateKey)
	require.ErrorContains(t, err, "not an SSH private key")
	_, statErr := os.Stat(privateKey + ".pub")
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}
