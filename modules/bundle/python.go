package bundle

import (
	"fmt"
	"os"
	"os/exec"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// PythonBundler manages Python tools via uv.
type PythonBundler struct {
	*baseBundler
}

// NewPythonBundler creates a new PythonBundler.
func NewPythonBundler(s store.Store) *PythonBundler {
	b := &PythonBundler{}
	pythonFile := pkgconfig.GetString("modules.bundle.python_file")
	if pythonFile == "" {
		pythonFile = "~/.config/cly/bundles/Pythonfile"
	}
	b.baseBundler = &baseBundler{
		name:        "python",
		defaultFile: pythonFile,
		store:       s,
		installFn:   b.install,
		uninstallFn: b.uninstall,
	}
	return b
}

func (b *PythonBundler) CheckDeps() error {
	if !commandExists("uv") {
		return fmt.Errorf("uv not found. Install: curl -LsSf https://astral.sh/uv/install.sh | sh")
	}
	return nil
}

func (b *PythonBundler) install(pkg string, verbose bool, force bool) error {
	cmd := exec.Command("uv", "tool", "install", "--force", "--upgrade", pkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uv tool install failed: %w", err)
	}
	return nil
}


func (b *PythonBundler) uninstall(pkg string, verbose bool) error {
	basePkg := extractBasePkg(pkg)

	cmd := exec.Command("uv", "tool", "uninstall", basePkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uv tool uninstall failed: %w", err)
	}
	return nil
}

// extractBasePkg extracts base package name from spec.
// vectorcode[lsp,mcp]<1.0.0 → vectorcode
// ruff → ruff
func extractBasePkg(pkg string) string {
	// Find first occurrence of [ < > = or @
	for i, c := range pkg {
		if c == '[' || c == '<' || c == '>' || c == '=' || c == '@' {
			return pkg[:i]
		}
	}
	return pkg
}
