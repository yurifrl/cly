package bundle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
		listInstalledFn: func() ([]string, error) {
			return listUvTools()
		},
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

// listUvTools runs `uv tool list` and returns installed base package names.
func listUvTools() ([]string, error) {
	cmd := exec.Command("uv", "tool", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("uv tool list failed: %w", err)
	}
	return parseUvToolList(string(output)), nil
}

// parseUvToolList parses `uv tool list` output.
// Format: "pkgname vX.Y.Z" lines for tools, "- binary" lines for executables.
func parseUvToolList(output string) []string {
	var tools []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "-") || line == "" {
			continue
		}
		// Line format: "pkgname vX.Y.Z"
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasPrefix(parts[1], "v") {
			tools = append(tools, parts[0])
		}
	}
	return tools
}

// diffByBaseName returns items from desired whose base names are not in installed.
// Packages with extras (containing '[') are always included since we can't verify
// if the correct extras are installed.
func diffByBaseName(desired, installed []string) []string {
	installedSet := make(map[string]bool)
	for _, pkg := range installed {
		installedSet[extractBasePkg(pkg)] = true
	}

	var result []string
	for _, pkg := range desired {
		hasExtras := strings.Contains(pkg, "[")
		baseInstalled := installedSet[extractBasePkg(pkg)]
		// Always reinstall if has extras (can't verify extras are correct)
		// Or if base name not installed at all
		if hasExtras || !baseInstalled {
			result = append(result, pkg)
		}
	}
	return result
}
