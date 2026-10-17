package bundle

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// GoBundler manages Go tools installed via `go install pkg@version`.
// Tool state is read from each binary's embedded build info, so the system
// itself is the source of truth and no bookkeeping store is needed.
type GoBundler struct {
	*baseBundler
}

// goTool is one installed Go tool with its embedded build info.
type goTool struct {
	binary  string // absolute path to the binary
	module  string // main module path from build info
	version string // module version ("(devel)" when built from source)
}

// NewGoBundler creates a new GoBundler.
func NewGoBundler() *GoBundler {
	b := &GoBundler{}
	goFile := pkgconfig.GetString("modules.bundle.go_file")
	if goFile == "" {
		goFile = "~/.config/cly/bundles/Gofile"
	}
	b.baseBundler = &baseBundler{
		name:        "go",
		defaultFile: goFile,
	}
	return b
}

func (b *GoBundler) CheckDeps() error {
	if !commandExists("go") {
		return fmt.Errorf("go not found. Add to Brewfile: 'go', then run: cly bundle brew")
	}
	return nil
}

// Sync installs, updates, and removes Go tools to match the Gofile.
func (b *GoBundler) Sync(bundleFile string, verbose bool, force bool, noUpdate bool, taps bool, mas bool) error {
	bundleFile = expandPath(bundleFile)
	lines, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	tools, err := listGoTools()
	if err != nil {
		return err
	}

	_, failed := removeUnwantedGoTools(lines, tools)

	// Decide what to install. Mirrors js semantics:
	// bare name installs once (upgrade with --upgrade), @latest always
	// refreshes, pinned versions reinstall on mismatch.
	var toInstall []string
	for _, line := range lines {
		spec, version := parseGoPkg(line)
		tool, found := findGoTool(spec, tools)
		switch {
		case !found:
			toInstall = append(toInstall, line)
		case version == "latest":
			// Floating spec — always refresh so it tracks upstream.
			toInstall = append(toInstall, line)
		case version == "":
			// Bare name — only reinstall on --upgrade or --force.
			if force || upgradeFlag {
				toInstall = append(toInstall, line)
			}
		case tool.version != version:
			// Pinned version mismatch — upgrade/downgrade.
			toInstall = append(toInstall, line)
		case force:
			toInstall = append(toInstall, line)
		}
	}

	if len(toInstall) > 0 {
		printGreen(fmt.Sprintf("Installing %d packages with go install...", len(toInstall)))
		for _, pkg := range toInstall {
			printYellow(fmt.Sprintf("Installing: %s", pkg))
			if err := b.install(pkg, verbose); err != nil {
				printRed(fmt.Sprintf("Warning: failed to install %s: %v", pkg, err))
				failed++
			}
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d package(s) failed", failed)
	}

	printGreen("\nDone!")
	return nil
}

// Check reports what would change without making changes.
func (b *GoBundler) Check(bundleFile string) error {
	bundleFile = expandPath(bundleFile)
	lines, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	tools, err := listGoTools()
	if err != nil {
		return err
	}

	var toInstall, toRemove []string
	for _, line := range lines {
		spec, version := parseGoPkg(line)
		tool, found := findGoTool(spec, tools)
		switch {
		case !found:
			toInstall = append(toInstall, fmt.Sprintf("%s (not installed)", line))
		case version == "latest":
			toInstall = append(toInstall, fmt.Sprintf("%s (installed: %s)", line, tool.version))
		case version == "":
			// Bare name, installed — only changes with --upgrade, so no-op here.
		case tool.version != version:
			toInstall = append(toInstall, fmt.Sprintf("%s (installed: %s)", line, tool.version))
		}
	}

	for _, tool := range tools {
		if !isWantedGoTool(tool, lines) {
			toRemove = append(toRemove, fmt.Sprintf("%s (%s@%s)", filepath.Base(tool.binary), tool.module, tool.version))
		}
	}

	if len(toInstall) == 0 && len(toRemove) == 0 {
		printGreen("Everything is in sync")
		return nil
	}

	if len(toInstall) > 0 {
		printGreen("Would install:")
		for _, pkg := range toInstall {
			fmt.Printf("  + %s\n", pkg)
		}
	}

	if len(toRemove) > 0 {
		printYellow("Would remove:")
		for _, pkg := range toRemove {
			fmt.Printf("  - %s\n", pkg)
		}
	}

	return fmt.Errorf("changes needed")
}

// Cleanup removes Go tools not listed in the Gofile.
func (b *GoBundler) Cleanup(bundleFile string, verbose bool, force bool) error {
	bundleFile = expandPath(bundleFile)
	lines, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	tools, err := listGoTools()
	if err != nil {
		return err
	}

	removed, failed := removeUnwantedGoTools(lines, tools)
	if removed == 0 && failed == 0 {
		printGreen("Nothing to clean up")
		return nil
	}

	printGreen("\nCleanup done!")
	return nil
}

// install runs `go install pkg@version`; bare specs default to @latest.
func (b *GoBundler) install(spec string, verbose bool) error {
	pkg, version := parseGoPkg(spec)
	if version == "" {
		version = "latest"
	}
	cmd := exec.Command("go", "install", pkg+"@"+version)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go install failed: %s", string(output))
	}
	return nil
}

// parseGoPkg splits an install spec into package path and version.
// The version separator is the last @: module paths never contain @.
// "golang.org/x/tools/cmd/goimports@v0.24.0" → ("golang.org/x/tools/cmd/goimports", "v0.24.0")
// "mvdan.cc/gofumpt@latest" → ("mvdan.cc/gofumpt", "latest")
// "mvdan.cc/gofumpt" → ("mvdan.cc/gofumpt", "")
func parseGoPkg(spec string) (string, string) {
	if idx := strings.LastIndex(spec, "@"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	return spec, ""
}

// matchGoTool reports whether an install spec refers to tool: the spec may
// name the tool's module itself or a main package inside it whose command
// name matches the binary (e.g. "m/cmd/goimports" → binary "goimports").
func matchGoTool(spec string, tool goTool) bool {
	if spec == tool.module {
		return true
	}
	return strings.HasPrefix(spec, tool.module+"/") && filepath.Base(spec) == filepath.Base(tool.binary)
}

// findGoTool returns the installed tool covered by an install spec, if any.
// spec may include an @version suffix, which is ignored for matching.
func findGoTool(spec string, tools []goTool) (goTool, bool) {
	spec, _ = parseGoPkg(spec)
	for _, tool := range tools {
		if matchGoTool(spec, tool) {
			return tool, true
		}
	}
	return goTool{}, false
}

// isWantedGoTool reports whether the Gofile lines cover tool.
func isWantedGoTool(tool goTool, lines []string) bool {
	for _, line := range lines {
		spec, _ := parseGoPkg(line)
		if matchGoTool(spec, tool) {
			return true
		}
	}
	return false
}

// removeUnwantedGoTools deletes installed tools not covered by the Gofile
// lines and returns the number of removals and failed removals.
func removeUnwantedGoTools(lines []string, tools []goTool) (removed, failed int) {
	for _, tool := range tools {
		if isWantedGoTool(tool, lines) {
			continue
		}
		printYellow(fmt.Sprintf("Removing: %s (%s@%s)", filepath.Base(tool.binary), tool.module, tool.version))
		if err := os.Remove(tool.binary); err != nil {
			printRed(fmt.Sprintf("Warning: failed to remove %s: %v", tool.binary, err))
			failed++
		} else {
			removed++
		}
	}
	return removed, failed
}

// listGoTools scans the Go binary directory and returns every executable
// carrying Go build info. Files without build info (non-Go binaries,
// scripts, stray files) are ignored — `go install` tools always embed it.
func listGoTools() ([]goTool, error) {
	out, err := exec.Command("go", "env", "GOBIN", "GOPATH").Output()
	if err != nil {
		return nil, fmt.Errorf("go env failed: %w", err)
	}
	lines := strings.SplitN(strings.TrimSuffix(string(out), "\n"), "\n", 2)
	if len(lines) != 2 {
		return nil, fmt.Errorf("unexpected `go env GOBIN GOPATH` output: %q", string(out))
	}
	dir := lines[0]
	if dir == "" {
		dir = filepath.Join(lines[1], "bin")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no bin dir yet: nothing installed
		}
		return nil, fmt.Errorf("failed to read %s: %w", dir, err)
	}

	var tools []goTool
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		info, err := os.Stat(path) // follows symlinks
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		build, err := buildinfo.ReadFile(path)
		if err != nil {
			continue // not a Go tool (no build info)
		}
		tools = append(tools, goTool{
			binary:  path,
			module:  build.Main.Path,
			version: build.Main.Version,
		})
	}
	return tools, nil
}
