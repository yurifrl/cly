package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPlatform(t *testing.T) {
	osName, arch := detectPlatform()

	assert.Contains(t, []string{"darwin", "linux"}, osName)
	assert.Contains(t, []string{"arm64", "amd64"}, arch)

	// Verify it matches runtime values
	assert.Equal(t, runtime.GOOS, osName)
	assert.Equal(t, runtime.GOARCH, arch)
}

func TestGetAssetName(t *testing.T) {
	tests := []struct {
		name string
		os   string
		arch string
		want string
	}{
		{
			name: "darwin arm64",
			os:   "darwin",
			arch: "arm64",
			want: "cly-darwin-arm64",
		},
		{
			name: "darwin amd64",
			os:   "darwin",
			arch: "amd64",
			want: "cly-darwin-amd64",
		},
		{
			name: "linux arm64",
			os:   "linux",
			arch: "arm64",
			want: "cly-linux-arm64",
		},
		{
			name: "linux amd64",
			os:   "linux",
			arch: "amd64",
			want: "cly-linux-amd64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAssetName(tt.os, tt.arch)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckLatest(t *testing.T) {
	// Read test fixture
	fixtureData, err := os.ReadFile("testdata/release_latest.json")
	require.NoError(t, err)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/yurifrl/cly/releases/latest", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixtureData)
	}))
	defer server.Close()

	// Create updater with test server
	currentVer, err := ParseVersion("v0.2.5")
	require.NoError(t, err)

	u := &Updater{
		repo:       "yurifrl/cly",
		currentVer: currentVer,
		httpClient: server.Client(),
		apiBaseURL: server.URL,
	}

	// Test CheckLatest
	release, err := u.CheckLatest()
	require.NoError(t, err)
	require.NotNil(t, release)

	assert.Equal(t, "v0.2.6", release.Version)
	assert.Len(t, release.Assets, 4)

	// Find darwin-arm64 asset
	var found bool
	for _, asset := range release.Assets {
		if asset.Name == "cly-darwin-arm64" {
			found = true
			assert.Equal(t, int64(5242880), asset.Size)
			assert.Contains(t, asset.DownloadURL, "https://github.com/yurifrl/cly/releases/download/v0.2.6/cly-darwin-arm64")
		}
	}
	assert.True(t, found, "darwin-arm64 asset not found")
}

func TestCheckLatest_NetworkError(t *testing.T) {
	// Create updater with invalid URL
	currentVer, err := ParseVersion("v0.2.5")
	require.NoError(t, err)

	u := &Updater{
		repo:       "yurifrl/cly",
		currentVer: currentVer,
		httpClient: &http.Client{},
		apiBaseURL: "http://invalid.url.that.does.not.exist",
	}

	// Should return error
	_, err = u.CheckLatest()
	assert.Error(t, err)
}

func TestFindAssetForPlatform(t *testing.T) {
	tests := []struct {
		name      string
		assets    []Asset
		os        string
		arch      string
		wantFound bool
		wantName  string
	}{
		{
			name: "asset found",
			assets: []Asset{
				{Name: "cly-darwin-arm64", DownloadURL: "https://example.com/cly-darwin-arm64"},
				{Name: "cly-linux-amd64", DownloadURL: "https://example.com/cly-linux-amd64"},
			},
			os:        "darwin",
			arch:      "arm64",
			wantFound: true,
			wantName:  "cly-darwin-arm64",
		},
		{
			name: "asset not found",
			assets: []Asset{
				{Name: "cly-linux-amd64", DownloadURL: "https://example.com/cly-linux-amd64"},
			},
			os:        "darwin",
			arch:      "arm64",
			wantFound: false,
		},
		{
			name:      "no assets",
			assets:    []Asset{},
			os:        "darwin",
			arch:      "arm64",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := &ReleaseInfo{
				Assets: tt.assets,
			}

			asset, found := release.FindAssetForPlatform(tt.os, tt.arch)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantName, asset.Name)
			}
		})
	}
}

func TestDownload(t *testing.T) {
	// Create test content
	testContent := []byte("fake binary content for testing")

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", string(rune(len(testContent))))
		w.Write(testContent)
	}))
	defer server.Close()

	// Create temp directory
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test-binary")

	// Create updater
	currentVer, err := ParseVersion("v0.2.5")
	require.NoError(t, err)

	u := &Updater{
		repo:       "yurifrl/cly",
		currentVer: currentVer,
		httpClient: server.Client(),
	}

	// Create test asset
	asset := Asset{
		Name:        "test-binary",
		DownloadURL: server.URL,
		Size:        int64(len(testContent)),
	}

	// Test download
	err = u.Download(asset, destPath)
	require.NoError(t, err)

	// Verify file exists and has correct content
	content, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Verify file is executable
	info, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0100 != 0, "file should be executable")
}

func TestInstall(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create fake current binary
	currentPath := filepath.Join(tmpDir, "cly")
	err := os.WriteFile(currentPath, []byte("old version"), 0755)
	require.NoError(t, err)

	// Create fake new binary
	newPath := filepath.Join(tmpDir, "cly-new")
	err = os.WriteFile(newPath, []byte("new version"), 0755)
	require.NoError(t, err)

	// Create updater
	currentVer, err := ParseVersion("v0.2.5")
	require.NoError(t, err)

	u := &Updater{
		repo:       "yurifrl/cly",
		currentVer: currentVer,
		httpClient: &http.Client{},
	}

	// Test install
	err = u.Install(newPath, currentPath)
	require.NoError(t, err)

	// Verify current binary has new content
	content, err := os.ReadFile(currentPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("new version"), content)

	// Verify backup was created and then removed
	backupPattern := filepath.Join(tmpDir, "cly.bak.*")
	matches, err := filepath.Glob(backupPattern)
	require.NoError(t, err)
	assert.Empty(t, matches, "backup should be cleaned up")
}

func TestInstall_WithRollback(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create fake current binary
	currentPath := filepath.Join(tmpDir, "cly")
	err := os.WriteFile(currentPath, []byte("old version"), 0755)
	require.NoError(t, err)

	// Don't create new binary to simulate failure

	// Create updater
	currentVer, err := ParseVersion("v0.2.5")
	require.NoError(t, err)

	u := &Updater{
		repo:       "yurifrl/cly",
		currentVer: currentVer,
		httpClient: &http.Client{},
	}

	// Test install with non-existent new binary
	newPath := filepath.Join(tmpDir, "nonexistent")
	err = u.Install(newPath, currentPath)
	assert.Error(t, err)

	// Verify current binary still has old content
	content, err := os.ReadFile(currentPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("old version"), content)
}

func TestNew(t *testing.T) {
	u := New("v0.2.5")
	require.NotNil(t, u)

	assert.Equal(t, "yurifrl/cly", u.repo)
	assert.NotNil(t, u.currentVer)
	assert.Equal(t, "v0.2.5", u.currentVer.Raw)
	assert.NotNil(t, u.httpClient)
	assert.Equal(t, "https://api.github.com", u.apiBaseURL)
}
