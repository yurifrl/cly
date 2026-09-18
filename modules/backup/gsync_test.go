package backup

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

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

func TestResolveSyncTarget_NoArgs(t *testing.T) {
	// no target falls through to the classic workdir backup without error,
	// whatever the user's live config says
	target, bucket, sources, err := resolveSyncTarget(nil)
	require.NoError(t, err)
	assert.Equal(t, "workdir", target)
	assert.Equal(t, getBucket(), bucket)
	require.Len(t, sources, 1)
	assert.Equal(t, getWorkdir(), sources[0].dir)
	assert.Empty(t, sources[0].prefix)
}

func TestResolveSyncTarget_RejectsPathyNames(t *testing.T) {
	for _, name := range []string{"a/b", ".", "..", `a\b`, "a.b", "/etc"} {
		_, _, _, err := resolveSyncTarget([]string{name})
		assert.Error(t, err, "target %q must be rejected", name)
	}
}

func TestResolveSyncTarget_Unconfigured(t *testing.T) {
	_, _, _, err := resolveSyncTarget([]string{"no-such-target-xyz"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no-such-target-xyz")
	assert.Contains(t, err.Error(), "not configured")
}

func TestObjectPrefixing(t *testing.T) {
	assert.Equal(t, "cly/main.go", objectName("", "cly/main.go"))
	assert.Equal(t, "omp/cly/main.go", objectName("omp", "cly/main.go"))
}

func TestDisplayFolderName(t *testing.T) {
	assert.Equal(t, "cly", displayFolderName("", "cly"))
	assert.Equal(t, "omp/cly", displayFolderName("omp", "cly"))
}

func TestSummarize(t *testing.T) {
	results := []folderResult{
		{uploaded: 2, skipped: 3, errors: 1},
		{uploaded: 5, skipped: 0, errors: 0},
	}
	s := summarize(results, "/tmp/report.md")
	assert.Equal(t, 2, s.folders)
	assert.Equal(t, 7, s.uploaded)
	assert.Equal(t, 3, s.skipped)
	assert.Equal(t, 1, s.errors)
	assert.Equal(t, "/tmp/report.md", s.reportPath)
}

func TestHistoryRoundtrip(t *testing.T) {
	// readHistory/appendHistory are file-based; point HOME at a temp dir so the
	// test never touches the real ~/.local/state/cly.
	home := t.TempDir()
	t.Setenv("HOME", home)

	entries, err := readHistory()
	require.NoError(t, err)
	assert.Empty(t, entries)

	now := time.Now()
	appendHistory(historyEntry{Time: now, Target: "sessions", Bucket: "bkt", Sources: 2, Folders: 13, Uploaded: 65, Skipped: 1, Errors: 0, Seconds: 28.5, Mode: "headless"})
	appendHistory(historyEntry{Time: now.Add(time.Minute), Target: "workdir", Bucket: "bkt2", Sources: 1, Folders: 40, Uploaded: 5, Skipped: 400, Errors: 2, Seconds: 90, Mode: "interactive"})

	entries, err = readHistory()
	require.NoError(t, err)
	require.Len(t, entries, 2)
	// oldest first
	assert.Equal(t, "sessions", entries[0].Target)
	assert.Equal(t, "workdir", entries[1].Target)
	assert.Equal(t, 65, entries[0].Uploaded)
	assert.Equal(t, 2, entries[1].Errors)
}
