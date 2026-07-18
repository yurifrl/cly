package backup

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExcludePatternMatches(t *testing.T) {
	re := regexp.MustCompile(buildExcludePattern())
	assert.True(t, re.MatchString("cly/node_modules/react/index.js"))
	assert.True(t, re.MatchString("cly/node_modules/")) // dir form used for SkipDir
	assert.True(t, re.MatchString("app/foo.pyc"))
	assert.True(t, re.MatchString("x/.DS_Store"))
	assert.False(t, re.MatchString("cly/src/main.go"))
	assert.False(t, re.MatchString("cly/README.md"))
}

func TestGlobToRegex(t *testing.T) {
	m := func(glob, path string) bool {
		return regexp.MustCompile(globToRegex(glob)).MatchString(path)
	}
	// directory glob matches the dir form and its contents, at any depth
	assert.True(t, m("node_modules/", "a/node_modules/"))
	assert.True(t, m("node_modules/", "a/node_modules/react/x.js"))
	assert.False(t, m("node_modules/", "a/renamed_modules/x"))
	// bare name matches the leaf, the dir form, and contents
	assert.True(t, m(".derivedData", "Yuri/Dayflow/.derivedData/x/v8.data"))
	assert.True(t, m(".derivedData", "Yuri/Dayflow/.derivedData/"))
	assert.True(t, m(".derivedData", "Yuri/.derivedData"))
	// extension glob
	assert.True(t, m("*.log", "a/b/foo.log"))
	assert.False(t, m("*.log", "a/b/foo.txt"))
	// leading-slash anchors to root
	assert.True(t, m("/build/", "build/out.o"))
	assert.False(t, m("/build/", "pkg/build/out.o"))
	// real files not matched
	assert.False(t, m("node_modules/", "src/main.go"))
}

func TestListWorkdirFolders(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "beta"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "alpha"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "loose.txt"), []byte("x"), 0o644))

	folders, loose, err := listWorkdirFolders(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"}, folders) // sorted, hidden excluded
	assert.Equal(t, []string{"loose.txt"}, loose)
}
