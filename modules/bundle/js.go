package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// JsBundler manages JavaScript packages via bun.
type JsBundler struct {
	*baseBundler
}

// NewJsBundler creates a new JsBundler.
func NewJsBundler(s store.Store) *JsBundler {
	b := &JsBundler{}
	jsFile := pkgconfig.GetString("modules.bundle.js_file")
	if jsFile == "" {
		jsFile = "~/.config/Jsfile"
	}
	b.baseBundler = &baseBundler{
		name:            "js",
		defaultFile:     jsFile,
		store:           s,
		installFn:       b.install,
		uninstallFn:     b.uninstall,
		listInstalledFn: b.ListInstalled,
	}
	return b
}

func (b *JsBundler) CheckDeps() error {
	if !commandExists("bun") {
		return fmt.Errorf("bun not found. Install: curl -fsSL https://bun.sh/install | bash")
	}
	return nil
}

func (b *JsBundler) install(pkg string, verbose bool, force bool) error {
	normalized := normalizePackage(pkg)

	args := []string{"install", "-g", normalized}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("bun", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if normalized != pkg {
			fmt.Printf("  → Resolved to: %s\n", normalized)
		}
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun install failed: %w", err)
	}
	return nil
}

// ListInstalled returns packages actually installed via bun pm ls -g
func (b *JsBundler) ListInstalled() ([]string, error) {
	cmd := exec.Command("bun", "pm", "ls", "-g", "--depth=0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	var packages []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Parse lines like "├── @fission-ai/openspec@0.16.0"
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "├──") || strings.HasPrefix(line, "└──") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Split package@version to get just package name
				pkgWithVersion := parts[1]
				pkg := strings.Split(pkgWithVersion, "@")
				if strings.HasPrefix(pkgWithVersion, "@") {
					// Scoped package like @foo/bar@1.0.0
					if len(pkg) >= 3 {
						packages = append(packages, "@"+pkg[1])
					}
				} else {
					// Regular package like foo@1.0.0
					if len(pkg) >= 1 {
						packages = append(packages, pkg[0])
					}
				}
			}
		}
	}
	return packages, nil
}

func (b *JsBundler) uninstall(pkg string, verbose bool) error {
	cmd := exec.Command("bun", "remove", "-g", pkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun remove failed: %w", err)
	}
	return nil
}

// normalizePackage converts GitHub shorthand to bun format.
// user/repo → github:user/repo
// user/repo@version → github:user/repo#version
// @scope/pkg → unchanged
var githubShorthand = regexp.MustCompile(`^([^@][^/]+)/([^@]+)(@.*)?$`)

func normalizePackage(pkg string) string {
	matches := githubShorthand.FindStringSubmatch(pkg)
	if matches == nil {
		// npm scoped packages or regular packages - pass through
		return pkg
	}

	user := matches[1]
	repo := matches[2]
	version := matches[3]

	// Convert @ version to # for tag/branch
	if version != "" {
		version = strings.Replace(version, "@", "#", 1)
	}

	return fmt.Sprintf("github:%s/%s%s", user, repo, version)
}
