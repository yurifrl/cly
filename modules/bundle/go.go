package bundle

import (
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

	// Remove tools not in desired
	_, failed := removeUnwantedGoTools(lines, tools)

	// Classify: installs for missing tools, upgrades/downgrades for version
	// moves, nothing for already-current entries.
	actions, examined, planErr := planGoSync(lines, tools, force)
	if planErr != nil {
		printYellow("Warning: " + planErr.Error())
	}

	if len(actions) > 0 {
		if parallelFlag && !ttyAvailable() {
			printYellow("No TTY available - running installs serially")
		}
		if parallelFlag && ttyAvailable() {
			// Parallel installs with TUI progress
			installFn := func(pkg string) (string, string, error) {
				note, err := b.install(pkg, verbose)
				return pkg, note, err
			}
			if err := runParallelInstall(installFn, actions); err != nil {
				return err
			}
		} else {
			// Default: one action at a time so each is visible
			for _, action := range actions {
				printYellow(action.label())
				note, err := b.install(action.pkg, verbose)
				if note != "" {
					printYellow(note)
				}
				if err != nil {
					printRed(fmt.Sprintf("Warning: failed to sync %s: %v", action.base, err))
					failed++
				}
			}
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d package(s) failed", failed)
	}

	printGreen("\n" + syncSummary(actions, examined))
	printGreen("Done!")
	return nil
}

// planGoSync classifies every Gofile entry against the installed tool
// inventory. Pinned entries compare locally (semver); bare entries and
// @latest float to the module proxy's latest. Entries the proxy cannot
// answer are left untouched and named in the returned error. Also returns
// the number of distinct bundle entries examined.
func planGoSync(desired []string, tools []goTool, force bool) ([]syncAction, int, error) {
	// Floating specs (bare name or @latest) need a proxy lookup — only for
	// tools that are already installed.
	toResolve := make(map[string]string) // spec -> original bundle line
	for _, line := range desired {
		spec, version := parseGoPkg(line)
		if _, seen := toResolve[spec]; seen {
			continue
		}
		if _, found := findGoTool(spec, tools); found && (version == "" || version == "latest") {
			toResolve[spec] = line
		}
	}
	latest, planErr := goLatestVersions(toResolve)

	var actions []syncAction
	seen := make(map[string]bool, len(desired))
	examined := 0
	for _, line := range desired {
		spec, version := parseGoPkg(line)
		if seen[spec] {
			continue
		}
		seen[spec] = true
		examined++
		base := filepath.Base(spec)
		tool, found := findGoTool(spec, tools)

		var verb, from, to string
		switch {
		case !found:
			verb, to = "install", version
		case force:
			// --force reinstalls everything; version moves keep their labels.
			if version == "" || version == "latest" || normalizeVersion(tool.version) == normalizeVersion(version) {
				verb = "reinstall"
			} else {
				verb, from, to = "upgrade", tool.version, version
				if tool.version != "(devel)" && compareVersions(tool.version, version) > 0 {
					verb = "downgrade"
				}
			}
		case version == "" || version == "latest":
			// Bare name / @latest tracks the proxy's latest; an unreachable
			// proxy means the installed version stays untouched.
			up, ok := latest[spec]
			if !ok {
				continue
			}
			if normalizeVersion(up) == normalizeVersion(tool.version) {
				continue
			}
			verb, from, to = "upgrade", tool.version, up
			if tool.version != "(devel)" && compareVersions(tool.version, up) > 0 {
				verb = "downgrade"
			}
		case normalizeVersion(tool.version) == normalizeVersion(version):
			continue
		default:
			// Pinned version moved - direction comes from the compare.
			verb, from, to = "upgrade", tool.version, version
			if tool.version != "(devel)" && compareVersions(tool.version, version) > 0 {
				verb = "downgrade"
			}
		}
		actions = append(actions, syncAction{pkg: line, base: base, verb: verb, from: from, to: to})
	}
	return actions, examined, planErr
}

// goLatestVersions resolves the upstream version for each spec through the
// module proxy (metadata requests only, no tarballs). Resolved entries are
// returned keyed by the original spec; failures are named in the error and
// simply absent from the map so callers can leave those packages untouched.
func goLatestVersions(specs map[string]string) (map[string]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(specs))
	for spec := range specs {
		keys = append(keys, spec)
	}
	sort.Strings(keys)
	client := &http.Client{Timeout: 10 * time.Second}
	return fetchConcurrent(keys, specs, func(spec string) (string, error) {
		v, err := goProxyLatest(client, spec)
		if err == nil {
			return v, nil
		}
		// The spec may be a main package inside its module
		// ("m/cmd/name"); retry against the owning module.
		if module := strings.TrimSuffix(spec, "/cmd/"+filepath.Base(spec)); module != spec {
			return goProxyLatest(client, module)
		}
		return "", err
	})
}

// goProxyLatest returns the module's latest version from the module proxy.
func goProxyLatest(client *http.Client, module string) (string, error) {
	url := goProxyURL() + "/" + escapeGoModule(module) + "/@latest"
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %s", url, resp.Status)
	}
	var body struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Version == "" {
		return "", fmt.Errorf("%s returned no version", url)
	}
	return body.Version, nil
}

// goProxyURL returns the module proxy to query: the first URL in GOPROXY,
// or the public default when unset, direct, or off.
func goProxyURL() string {
	def := "https://proxy.golang.org"
	gp := os.Getenv("GOPROXY")
	if gp == "" {
		return def
	}
	first, _, _ := strings.Cut(gp, ",")
	if first == "" || first == "direct" || first == "off" {
		return def
	}
	return strings.TrimSuffix(first, "/")
}

// escapeGoModule encodes a module path for the module proxy: uppercase
// letters are escaped as '!' plus the lowercase letter.
func escapeGoModule(module string) string {
	var b strings.Builder
	for _, r := range module {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Check reports drift between the Gofile and the real tool state using the
// same classification as Sync, including proxy-backed checks for bare and
// @latest entries.
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

	actions, examined, planErr := planGoSync(lines, tools, false)
	if planErr != nil {
		printYellow("Warning: " + planErr.Error())
	}

	var extras []string
	for _, tool := range tools {
		if !isWantedGoTool(tool, lines) {
			extras = append(extras, fmt.Sprintf("%s (%s@%s)", filepath.Base(tool.binary), tool.module, tool.version))
		}
	}
	sort.Strings(extras)

	if len(actions) == 0 && len(extras) == 0 {
		printGreen(fmt.Sprintf("Everything is in sync - %d tools up to date", examined))
		return nil
	}
	if len(actions) > 0 {
		printGreen("Would change:")
		for _, action := range actions {
			fmt.Printf("  %s\n", action.label())
		}
	}
	if len(extras) > 0 {
		printYellow("Would remove:")
		for _, pkg := range extras {
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

// goFallbackEnv returns the fallback environment (space-separated KEY=VALUE
// pairs) configured in the global config; empty means no fallback and failed
// installs fail as-is.
func goFallbackEnv() []string {
	var env []string
	for _, kv := range strings.Fields(pkgconfig.GetString("modules.bundle.go_fallback_env")) {
		if strings.Contains(kv, "=") {
			env = append(env, kv)
		}
	}
	return env
}

// goInstallWithFallback runs `go install spec` unchanged; when it fails and a
// fallback env is configured, it retries once with those environment
// overrides (e.g. GOPROXY=direct GOSUMDB=sum.golang.org for a proxy that
// rejects fresh packages). The returned note is non-empty whenever the
// fallback was used, so callers can surface it to the user.
func goInstallWithFallback(spec string) (note string, err error) {
	_, err = runGoInstall(spec, nil)
	if err == nil {
		return "", nil
	}
	fallback := goFallbackEnv()
	if len(fallback) == 0 {
		return "", err
	}
	_, retryErr := runGoInstall(spec, fallback)
	if retryErr == nil {
		return fmt.Sprintf("%s failed on the default GOPROXY, installed via fallback env %s", spec, strings.Join(fallback, " ")), nil
	}
	return "", fmt.Errorf("%w (fallback env %s also failed: %v)", err, strings.Join(fallback, " "), retryErr)
}

// runGoInstall runs `go install spec`, appending env overrides on top of the
// ambient environment when set. os/exec keeps the last value for duplicate
// keys, so appended overrides win.
func runGoInstall(spec string, env []string) (string, error) {
	cmd := exec.Command("go", "install", spec)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("go install failed: %s", strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// install runs `go install pkg@version`; bare specs default to @latest.
// Returns any fallback note so callers can surface it.
func (b *GoBundler) install(spec string, verbose bool) (string, error) {
	pkg, version := parseGoPkg(spec)
	if version == "" {
		version = "latest"
	}
	return goInstallWithFallback(pkg + "@" + version)
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
