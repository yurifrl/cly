package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// GoBundler manages Go binaries.
type GoBundler struct {
	*baseBundler
	gobin string
}

// NewGoBundler creates a new GoBundler.
func NewGoBundler(s store.Store) *GoBundler {
	b := &GoBundler{}
	goFile := pkgconfig.GetString("bundle.go_file")
	if goFile == "" {
		goFile = "~/.config/Gofile"
	}
	b.baseBundler = &baseBundler{
		name:        "go",
		defaultFile: goFile,
		store:       s,
		installFn:   b.install,
		uninstallFn: b.uninstall,
	}
	return b
}

func (b *GoBundler) CheckDeps() error {
	if !commandExists("go") {
		return fmt.Errorf("go not found. Install Go: https://go.dev/dl/")
	}

	// Detect GOBIN - prefer mise if available
	b.gobin = b.detectGobin()
	return nil
}

func (b *GoBundler) detectGobin() string {
	// Try mise first
	if commandExists("mise") {
		cmd := exec.Command("mise", "where", "go")
		out, err := cmd.Output()
		if err == nil {
			miseGoRoot := strings.TrimSpace(string(out))
			if miseGoRoot != "" {
				return filepath.Join(miseGoRoot, "bin")
			}
		}
	}

	// Fallback to go env GOPATH
	cmd := exec.Command("go", "env", "GOPATH")
	out, err := cmd.Output()
	if err == nil {
		gopath := strings.TrimSpace(string(out))
		if gopath != "" {
			return filepath.Join(gopath, "bin")
		}
	}

	// Last resort
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

func (b *GoBundler) install(pkg string, verbose bool, force bool) error {
	// Set GOBIN for mise integration
	env := os.Environ()
	env = append(env, "GOBIN="+b.gobin)

	// Strip existing version suffix if present
	if idx := strings.Index(pkg, "@"); idx != -1 {
		pkg = pkg[:idx]
	}

	args := []string{"install", pkg + "@latest"}
	cmd := exec.Command("go", args...)
	cmd.Env = env

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}
	return nil
}

func (b *GoBundler) uninstall(pkg string, verbose bool) error {
	binaryName := getBinaryName(pkg)
	binaryPath := filepath.Join(b.gobin, binaryName)

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		if verbose {
			fmt.Printf("Binary %s not found, skipping\n", binaryPath)
		}
		return nil
	}

	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", binaryPath, err)
	}

	if verbose {
		fmt.Printf("Removed %s\n", binaryPath)
	}
	return nil
}

// getBinaryName extracts binary name from package path.
// github.com/foo/bar/cmd/baz → baz
// github.com/foo/bar → bar
func getBinaryName(pkg string) string {
	// Remove version suffix if present
	if idx := strings.Index(pkg, "@"); idx != -1 {
		pkg = pkg[:idx]
	}

	parts := strings.Split(pkg, "/")

	// Check for /cmd/ pattern
	for i, part := range parts {
		if part == "cmd" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	// Default to last segment
	return parts[len(parts)-1]
}
