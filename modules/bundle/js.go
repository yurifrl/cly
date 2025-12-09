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
	jsFile := pkgconfig.GetString("bundle.js_file")
	if jsFile == "" {
		jsFile = "~/.config/Jsfile"
	}
	b.baseBundler = &baseBundler{
		name:        "js",
		defaultFile: jsFile,
		store:       s,
		installFn:   b.install,
		uninstallFn: b.uninstall,
	}
	return b
}

func (b *JsBundler) CheckDeps() error {
	if !commandExists("bun") {
		return fmt.Errorf("bun not found. Install: curl -fsSL https://bun.sh/install | bash")
	}
	return nil
}

func (b *JsBundler) install(pkg string, verbose bool) error {
	normalized := normalizePackage(pkg)

	args := []string{"install", "-g", normalized}
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
