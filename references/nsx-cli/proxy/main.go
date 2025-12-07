package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	GitHubRawBaseURL   = "https://raw.githubusercontent.com/NSXBet/nsx-cli"
	GitHubAPIBaseURL   = "https://api.github.com/repos/NSXBet/nsx-cli"
	GithubGistsBaseURL = "https://raw.githubusercontent.com/NSXBet/gists"
)

type Config struct {
	GithubToken string
	Port        int
}

type Release struct {
	TagName   string    `json:"tag_name"`
	CreatedAt time.Time `json:"created_at"`
	Assets    []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"assets"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

func NewProxy(config Config) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: createHandler(config.GithubToken),
	}
}

func createHandler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		switch {
		case path == "releases":
			handleReleases(w, r, token)
		case path == "latest":
			handleLatestRelease(w, r, token)
		case strings.HasPrefix(path, "releases/download/"):
			handleReleaseDownload(w, r, path, token)
		case strings.HasPrefix(path, "gists/"):
			handleGistFile(w, r, path, token)
		default:
			handleRawFile(w, r, path, token)
		}
	}
}

func handleReleases(w http.ResponseWriter, _ *http.Request, token string) {
	url := GitHubAPIBaseURL + "/releases"
	resp, err := makeGitHubRequest(url, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func handleLatestRelease(w http.ResponseWriter, _ *http.Request, token string) {
	url := GitHubAPIBaseURL + "/releases/latest"
	resp, err := makeGitHubRequest(url, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func handleReleaseDownload(w http.ResponseWriter, _ *http.Request, path, token string) {
	parts := strings.Split(path, "/")
	if len(parts) != 4 {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	version, filename := parts[2], parts[3]
	url := fmt.Sprintf("%s/releases/tags/%s", GitHubAPIBaseURL, version)

	resp, err := makeGitHubRequest(url, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "release not found", http.StatusNotFound)
		return
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		http.Error(w, "failed to parse release data", http.StatusInternalServerError)
		return
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == filename {
			downloadURL = asset.URL
			break
		}
	}

	if downloadURL == "" {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}

	assetResp, err := makeGitHubDownloadRequest(downloadURL, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer assetResp.Body.Close()

	for key, values := range assetResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	io.Copy(w, assetResp.Body)
}

func makeGitHubRequest(url, token string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("error creating request", err)
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	return client.Do(req)
}

func makeGitHubDownloadRequest(url, token string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("error creating request", err)
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	log.Println("original url", url)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/octet-stream")

	client := &http.Client{}
	return client.Do(req)
}

func handleRawFile(w http.ResponseWriter, _ *http.Request, path, token string) {
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("%s/%s", GitHubRawBaseURL, path)
	if resp, err := makeGitHubRequest(url, token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Println("github request failed", resp.StatusCode)
			http.Error(w, "github request failed", resp.StatusCode)
			return
		}

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		io.Copy(w, resp.Body)
	}
}

func handleGistFile(w http.ResponseWriter, _ *http.Request, path, token string) {
	// strip gists/ from path
	path = strings.TrimPrefix(path, "gists/")
	log.Println("path", path)

	url := fmt.Sprintf("%s/%s", GithubGistsBaseURL, path)
	if resp, err := makeGitHubRequest(url, token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Println("github request failed", resp.StatusCode)
			http.Error(w, "github request failed", resp.StatusCode)
			return
		}

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		io.Copy(w, resp.Body)
	}
}

func main() {
	var (
		githubToken = os.Getenv("GITHUB_TOKEN")
		config      = Config{GithubToken: githubToken, Port: 8080}
		server      = NewProxy(config)
	)

	log.Fatal(server.ListenAndServe())
}
