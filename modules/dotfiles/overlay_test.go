package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withUserOverride(t *testing.T, name string) {
	t.Helper()
	previous := userFlag
	userFlag = name
	t.Cleanup(func() { userFlag = previous })
}

func withConfigFlag(t *testing.T, path string) {
	t.Helper()
	previous := configFlag
	configFlag = path
	t.Cleanup(func() { configFlag = previous })
}

func writeConf(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func destinations(cfg *Config) []string {
	out := make([]string, 0, len(cfg.Mappings))
	for _, mapping := range cfg.Mappings {
		out = append(out, mapping.Destination)
	}
	return out
}

func TestLoadConfig_BaseAlwaysAppliedWithUserOverlay(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "./base -> "+dir+"/base.out\n")
	writeConf(t, filepath.Join(dir, "dotfiles.bob.conf"), "./extra -> "+dir+"/extra.out\n")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	cfg, paths, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "dotfiles.conf"),
		filepath.Join(dir, "dotfiles.bob.conf"),
	}, paths, "base config must always load and the user config is an overlay")
	assert.Equal(t, []string{dir + "/base.out", dir + "/extra.out"}, destinations(cfg))
}

func TestLoadConfig_OverlayAbsentLoadsBaseOnly(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "./base -> "+dir+"/base.out\n")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "nobody", "linux", "amd64")

	cfg, paths, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "dotfiles.conf")}, paths)
	assert.Equal(t, []string{dir + "/base.out"}, destinations(cfg))
}

func TestLoadConfig_UserFlagSelectsOverlayAndTargetUser(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "./base -> "+dir+"/base.out\n")
	writeConf(t, filepath.Join(dir, "dotfiles.alice.conf"), "@target user=alice\n./alice -> "+dir+"/alice.out\n")
	writeConf(t, filepath.Join(dir, "dotfiles.bob.conf"), "@target user=bob\n./bob -> "+dir+"/bob.out\n")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")
	withUserOverride(t, "alice")

	cfg, paths, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "dotfiles.conf"),
		filepath.Join(dir, "dotfiles.alice.conf"),
	}, paths, "--user must select the overlay rather than the detected user")
	assert.Equal(t, []string{dir + "/base.out", dir + "/alice.out"}, destinations(cfg))
}

func TestLoadConfig_NonMatchingOverlayTargetSkipsOnlyOverlay(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "./base -> "+dir+"/base.out\n")
	writeConf(t, filepath.Join(dir, "dotfiles.bob.conf"), "@target arch=arm64\n./arm -> "+dir+"/arm.out\n")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	cfg, paths, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "dotfiles.conf")}, paths,
		"an arch-gated overlay must not prevent the base config from applying")
	assert.Equal(t, []string{dir + "/base.out"}, destinations(cfg))
	require.Len(t, cfg.Skipped, 1)
	assert.Contains(t, cfg.Skipped[0], "dotfiles.bob.conf")
}

func TestLoadConfig_NonMatchingBaseTargetIsAnError(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "@target os=windows\n./win -> "+dir+"/win.out\n")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	_, _, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no dotfiles config matches this machine")
}

func TestLoadConfig_ExplicitUserConfigAlsoLoadsSiblingBase(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "dotfiles.conf")
	overlay := filepath.Join(dir, "dotfiles.alice.conf")
	writeConf(t, base, "./base -> "+dir+"/base.out\n")
	writeConf(t, overlay, "./alice -> "+dir+"/alice.out\n")

	withConfigFlag(t, overlay)
	withContext(t, "bob", "linux", "amd64")

	cfg, paths, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{base, overlay}, paths)
	assert.Equal(t, []string{dir + "/base.out", dir + "/alice.out"}, destinations(cfg))
}

func TestLoadConfig_MissingExplicitConfigErrors(t *testing.T) {
	dir := t.TempDir()
	withConfigFlag(t, filepath.Join(dir, "nonexistent.conf"))
	withContext(t, "bob", "linux", "amd64")

	_, _, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config not found")
}

func TestLoadConfig_ErrorsArePrefixedWithFilename(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "./ok -> "+dir+"/ok.out\n")
	writeConf(t, filepath.Join(dir, "dotfiles.bob.conf"), "@bogus thing\n")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	cfg, _, err := loadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.Errors, 1)
	assert.Contains(t, cfg.Errors[0], "dotfiles.bob.conf")
	assert.Contains(t, cfg.Errors[0], "unknown directive")
}

func TestLockFilePath_IsAlwaysTheBaseLock(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "")
	writeConf(t, filepath.Join(dir, "dotfiles.bob.conf"), "")

	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	got, err := lockFilePath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "dotfiles.lock"), got)
}

func TestLoadLock_AdoptsPerUserLockEntries(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "")
	writeConf(t, filepath.Join(dir, "dotfiles.bob.conf"), "")
	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	basePath := filepath.Join(dir, "dotfiles.lock")
	userPath := filepath.Join(dir, "dotfiles.bob.lock")
	writeConf(t, basePath, `{"symlinks":[{"source":"/s/a","destination":"/d/a"}],"install_commands":["echo base"]}`)
	writeConf(t, userPath, `{"symlinks":[{"source":"/s/b","destination":"/d/b"}],"install_commands":["echo bob"]}`)

	lock, err := loadLock(basePath)
	require.NoError(t, err)

	var dests []string
	for _, e := range lock.Symlinks {
		dests = append(dests, e.Destination)
	}
	assert.ElementsMatch(t, []string{"/d/a", "/d/b"}, dests,
		"per-user lock entries must be adopted so their artifacts stay tracked for cleanup")
	assert.ElementsMatch(t, []string{"echo base", "echo bob"}, lock.InstallCommands)

	_, statErr := os.Stat(userPath)
	assert.True(t, os.IsNotExist(statErr), "per-user lock is consumed once and removed")
}

func TestLoadLock_AdoptionDoesNotDuplicateEntries(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, filepath.Join(dir, "dotfiles.conf"), "")
	withConfigFlag(t, filepath.Join(dir, "dotfiles.conf"))
	withContext(t, "bob", "linux", "amd64")

	basePath := filepath.Join(dir, "dotfiles.lock")
	userPath := filepath.Join(dir, "dotfiles.bob.lock")
	writeConf(t, basePath, `{"symlinks":[{"source":"/s/a","destination":"/d/a"}]}`)
	writeConf(t, userPath, `{"symlinks":[{"source":"/s/a","destination":"/d/a"}]}`)

	lock, err := loadLock(basePath)
	require.NoError(t, err)
	assert.Len(t, lock.Symlinks, 1, "the same destination must not be tracked twice")
}
