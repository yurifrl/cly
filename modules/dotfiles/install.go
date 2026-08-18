package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/mut"
	"github.com/yurifrl/cly/pkg/style"
)

type InstallOptions struct {
	Reinstall bool
	FailFast  bool
}

func ApplyInstalls(cfg *Config, opts InstallOptions) error {
	if len(cfg.Installs) == 0 {
		return nil
	}

	lockFile, err := lockFilePath()
	if err != nil {
		return err
	}
	lock, err := loadLock(lockFile)
	if err != nil {
		return err
	}

	existing := make(map[string]InstallManifest, len(lock.Installs))
	for _, e := range lock.Installs {
		existing[e.URL] = e
	}

	for _, inst := range cfg.Installs {
		prev, known := existing[inst.URL]

		sha, scriptPath, err := fetchAndCacheScript(inst.URL)
		if err != nil {
			fmt.Printf("  %s fetch %s: %s\n", style.RedStyle.Render("❌"), inst.URL, err)
			if opts.FailFast {
				return err
			}
			continue
		}

		if known && prev.SHA == sha && !opts.Reinstall {
			fmt.Printf("  %s @install %s (up to date)\n", style.SubtleStyle.Render("○"), inst.URL)
			continue
		}

		if mut.DryRun() {
			fmt.Printf("  %s [dry-run] @install %s\n", style.YellowStyle.Render("⊘"), inst.URL)
			continue
		}

		if err := runScript(scriptPath, cfg.BaseDir); err != nil {
			fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
			if opts.FailFast {
				return err
			}
			continue
		}

		lock.Installs = upsertInstall(lock.Installs, InstallManifest{URL: inst.URL, SHA: sha})
		_ = saveLock(lockFile, lock)
		fmt.Printf("  %s @install %s\n", style.GreenStyle.Render("✅"), inst.URL)
	}

	return nil
}

func fetchAndCacheScript(url string) (sha, path string, err error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("fetch: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read: %w", err)
	}

	sum := sha256.Sum256(body)
	sha = hex.EncodeToString(sum[:])

	cacheDir := installCacheDir()
	if err := mut.MkdirAll(cacheDir, 0755); err != nil {
		return "", "", fmt.Errorf("cache dir: %w", err)
	}
	path = filepath.Join(cacheDir, sha+".sh")
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		if err := mut.WriteFile(path, body, 0755); err != nil {
			return "", "", fmt.Errorf("cache write: %w", err)
		}
	}
	return sha, path, nil
}

func installCacheDir() string {
	dataDir := pkgconfig.GetString("app.data_dir")
	if dataDir == "" {
		dataDir = "~/.local/share/cly"
	}
	return filepath.Join(expandTilde(dataDir), "dotfiles/install-cache")
}

// RemoveInstallArtifacts cleans up installed files for a removed @install entry.
func RemoveInstallArtifacts(e InstallManifest) {
	if e.Bypassed || e.Manifest == nil {
		printInstallCleanupBanner(e.URL, nil)
		return
	}
	m := e.Manifest
	for _, b := range m.Binaries {
		p := expandTilde(b)
		if mut.Remove(p) == nil {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed binary:"), shortenPath(p))
		}
	}
	for _, f := range m.Files {
		p := expandTilde(f)
		if mut.Remove(p) == nil {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed file:"), shortenPath(p))
		}
	}
	for _, d := range m.Dirs {
		p := expandTilde(d)
		entries, err := os.ReadDir(p)
		if err == nil && len(entries) == 0 {
			if mut.Remove(p) == nil {
				fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed dir:"), shortenPath(p))
			}
		}
	}
	if len(m.ShellRCChanges) > 0 {
		printInstallCleanupBanner(e.URL, m.ShellRCChanges)
	}
}

func printInstallCleanupBanner(url string, shellChanges []string) {
	sep := style.RedStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n%s\n", sep)
	if shellChanges == nil {
		fmt.Printf("%s\n", style.RedStyle.Render("  ⛔  REMOVED @install (no manifest) — manual cleanup required"))
		fmt.Printf("  URL: %s\n", url)
	} else {
		fmt.Printf("%s\n", style.RedStyle.Render("  ⛔  REMOVED @install — shell RC changes require manual cleanup"))
		fmt.Printf("  URL: %s\n", url)
		for _, line := range shellChanges {
			fmt.Printf("  %s  %s\n", style.RedStyle.Render("▶"), line)
		}
	}
	fmt.Printf("%s\n\n", sep)
}

func upsertInstall(entries []InstallManifest, e InstallManifest) []InstallManifest {
	for i, existing := range entries {
		if existing.URL == e.URL {
			entries[i] = e
			return entries
		}
	}
	return append(entries, e)
}
