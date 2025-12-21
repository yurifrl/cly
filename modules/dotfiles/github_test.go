package dotfiles

import (
	"os"
	"testing"
)

func TestParseGithubURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    githubRepo
		wantErr bool
	}{
		{
			name: "https with www",
			url:  "https://www.github.com/owner/repo",
			want: githubRepo{owner: "owner", repo: "repo"},
		},
		{
			name: "https without www",
			url:  "https://github.com/owner/repo",
			want: githubRepo{owner: "owner", repo: "repo"},
		},
		{
			name: "http",
			url:  "http://github.com/owner/repo",
			want: githubRepo{owner: "owner", repo: "repo"},
		},
		{
			name: "with trailing slash",
			url:  "https://github.com/owner/repo/",
			want: githubRepo{owner: "owner", repo: "repo"},
		},
		{
			name: "with .git suffix",
			url:  "https://github.com/owner/repo.git",
			want: githubRepo{owner: "owner", repo: "repo"},
		},
		{
			name:    "invalid domain",
			url:     "https://gitlab.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "missing owner",
			url:     "https://github.com/repo",
			wantErr: true,
		},
		{
			name:    "empty",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGithubURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGithubURL() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("parseGithubURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.owner != tt.want.owner || got.repo != tt.want.repo {
				t.Errorf("parseGithubURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildReleaseURL(t *testing.T) {
	repo := githubRepo{owner: "owner", repo: "repo"}
	want := "https://github.com/owner/repo/releases/latest/download/repo.wasm"
	got := buildReleaseURL(repo)
	if got != want {
		t.Errorf("buildReleaseURL() = %v, want %v", got, want)
	}
}

func TestDownloadZellijPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	err := downloadZellijPlugin("https://github.com/dj95/zjstatus")
	if err != nil {
		t.Fatalf("downloadZellijPlugin() error = %v", err)
	}

	expectedPath := tempDir + "/.config/zellij/plugins/zjstatus.wasm"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected file not created: %s", expectedPath)
	}
}
