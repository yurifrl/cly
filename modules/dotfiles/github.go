package dotfiles

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

type githubRepo struct {
	owner string
	repo  string
}

func parseGithubURL(url string) (githubRepo, error) {
	if url == "" {
		return githubRepo{}, fmt.Errorf("empty URL")
	}

	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "www.")
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")

	if !strings.HasPrefix(url, "github.com/") {
		return githubRepo{}, fmt.Errorf("not a github.com URL")
	}

	path := strings.TrimPrefix(url, "github.com/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		return githubRepo{}, fmt.Errorf("invalid GitHub URL format, expected github.com/owner/repo")
	}

	return githubRepo{
		owner: parts[0],
		repo:  parts[1],
	}, nil
}

func buildReleaseURL(repo githubRepo) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s.wasm", repo.owner, repo.repo, repo.repo)
}

func downloadZellijPlugin(githubURL string) error {
	repo, err := parseGithubURL(githubURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub URL: %w", err)
	}

	downloadURL := buildReleaseURL(repo)

	destDir := pkgconfig.GetString("modules.dotfiles.zellij_plugins_dir")
	if destDir == "" {
		destDir = "~/.config/zellij/plugins"
	}
	destDir = expandTilde(destDir)
	destPath := filepath.Join(destDir, repo.repo+".wasm")

	if dryRun {
		logDry("download", fmt.Sprintf("%s -> %s", downloadURL, destPath))
		return nil
	}

	if err := mutMkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d from %s", resp.StatusCode, downloadURL)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("  Downloaded %s.wasm to %s\n", repo.repo, destPath)
	return nil
}
