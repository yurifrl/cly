package dotfiles

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/llm"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/mut"
	"github.com/yurifrl/cly/pkg/style"
)

// newInstallLLMClient can be overridden in tests.
var newInstallLLMClient = func() (llm.Client, error) {
	return llm.NewClient(llm.Config{
		Provider: llm.ProviderAnthropic,
		Model:    "claude-haiku-4-5-20251001",
	})
}

type InstallOptions struct {
	BypassAI bool
	FailFast bool
}

type analysisResult struct {
	Manifest      ScriptManifest `json:"manifest"`
	Security      struct {
		Risk     string   `json:"risk"`
		Findings []string `json:"findings"`
	} `json:"security"`
	Final         bool   `json:"final"`
	MessageToUser string `json:"message_to_user"`
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

	var client llm.Client

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

		if known && prev.SHA == sha {
			fmt.Printf("  %s @install %s (up to date)\n", style.SubtleStyle.Render("○"), inst.URL)
			continue
		}

		if mut.DryRun() {
			fmt.Printf("  %s [dry-run] @install %s\n", style.YellowStyle.Render("⊘"), inst.URL)
			continue
		}

		if opts.BypassAI {
			fmt.Printf("  %s @install %s (bypass-ai, no manifest)\n", style.YellowStyle.Render("⚠️ "), inst.URL)
			if err := runScript(scriptPath, cfg.BaseDir); err != nil {
				fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
				if opts.FailFast {
					return err
				}
				continue
			}
			lock.Installs = upsertInstall(lock.Installs, InstallManifest{URL: inst.URL, SHA: sha, Bypassed: true})
			_ = saveLock(lockFile, lock)
			continue
		}

		if client == nil {
			client, err = newInstallLLMClient()
			if err != nil {
				fmt.Printf("  %s LLM unavailable (%s) — use --bypass-ai to skip analysis\n", style.RedStyle.Render("❌"), err)
				if opts.FailFast {
					return err
				}
				continue
			}
		}

		scriptBytes, _ := os.ReadFile(scriptPath)
		analysis, ok := analyzeScript(client, inst.URL, string(scriptBytes))

		var entry InstallManifest
		if ok {
			printInstallAnalysis(inst.URL, analysis)
			fmt.Print("Approve? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			if input := strings.ToLower(strings.TrimSpace(line)); input != "y" && input != "yes" {
				fmt.Printf("  %s @install %s aborted\n", style.YellowStyle.Render("⊘"), inst.URL)
				continue
			}
			m := analysis.Manifest
			entry = InstallManifest{URL: inst.URL, SHA: sha, Manifest: &m}
		} else {
			fmt.Printf("  %s @install %s: LLM analysis failed, installing without manifest\n", style.YellowStyle.Render("⚠️ "), inst.URL)
			entry = InstallManifest{URL: inst.URL, SHA: sha, Bypassed: true}
		}

		if err := runScript(scriptPath, cfg.BaseDir); err != nil {
			fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
			if opts.FailFast {
				return err
			}
			continue
		}

		lock.Installs = upsertInstall(lock.Installs, entry)
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

var installSystemPrompt = `You analyze install scripts and produce a removal manifest.
Output raw JSON only. Do not use markdown, code fences, or prose. Start your response with { and end with }.`

var installAnalysisPrompt = `Analyze this install script from %s:

` + "```sh\n%s\n```" + `

Return JSON:
{
  "manifest": {
    "binaries": ["absolute or ~ paths of installed binaries"],
    "dirs": ["created directories"],
    "files": ["other created files"],
    "shell_rc_changes": ["lines added to shell RC files"],
    "fetches_other_scripts": false,
    "needs_sudo": false
  },
  "security": {
    "risk": "low|medium|high",
    "findings": ["any concerns"]
  },
  "final": false,
  "message_to_user": "one-sentence summary"
}`

func analyzeScript(client llm.Client, url, script string) (analysisResult, bool) {
	prompt := fmt.Sprintf(installAnalysisPrompt, url, script)
	raw, err := client.Complete(context.Background(), installSystemPrompt, []llm.Message{{Role: llm.RoleUser, Content: prompt}})
	if err != nil {
		fmt.Printf("  %s LLM error: %s\n", style.RedStyle.Render("❌"), err)
		return analysisResult{}, false
	}
	var result analysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		if verboseFlag {
			fmt.Printf("\n%s\n", style.SubtleStyle.Render(raw))
		}
		return analysisResult{}, false
	}
	return result, true
}

func printInstallAnalysis(url string, a analysisResult) {
	riskColor := style.GreenStyle
	if a.Security.Risk == "medium" {
		riskColor = style.YellowStyle
	} else if a.Security.Risk == "high" {
		riskColor = style.RedStyle
	}

	fmt.Printf("\n%s %s\n", style.BlueStyle.Render("Script:"), url)
	if a.Security.Risk != "" {
		fmt.Printf("%s %s\n", style.SubtleStyle.Render("Risk:"), riskColor.Render(strings.ToUpper(a.Security.Risk)))
	}
	for _, f := range a.Security.Findings {
		fmt.Printf("  %s %s\n", style.YellowStyle.Render("⚠️ "), f)
	}

	m := a.Manifest
	if len(m.Binaries) > 0 {
		fmt.Printf("%s %s\n", style.SubtleStyle.Render("Binaries:"), strings.Join(m.Binaries, ", "))
	}
	if len(m.Dirs) > 0 {
		fmt.Printf("%s %s\n", style.SubtleStyle.Render("Dirs:"), strings.Join(m.Dirs, ", "))
	}
	if len(m.Files) > 0 {
		fmt.Printf("%s %s\n", style.SubtleStyle.Render("Files:"), strings.Join(m.Files, ", "))
	}
	if len(m.ShellRCChanges) > 0 {
		fmt.Printf("%s (manual cleanup if removed)\n", style.SubtleStyle.Render("Shell RC:"))
		for _, rc := range m.ShellRCChanges {
			fmt.Printf("  %s\n", rc)
		}
	}
	if m.FetchesOther {
		fmt.Printf("  %s fetches additional scripts\n", style.YellowStyle.Render("⚠️ "))
	}
	if m.NeedsSudo {
		fmt.Printf("  %s requires sudo\n", style.YellowStyle.Render("⚠️ "))
	}
	if a.MessageToUser != "" {
		fmt.Printf("%s %s\n", style.SubtleStyle.Render("Summary:"), a.MessageToUser)
	}
	fmt.Println()
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
