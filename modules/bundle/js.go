package bundle

import (
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
	"github.com/yurifrl/cly/pkg/store"
)

// JsBundler manages JavaScript packages via pnpm or npm.
type JsBundler struct {
	*baseBundler
	manager string // "pnpm" or "npm"
}

// NewJsBundler creates a new JsBundler.
func NewJsBundler(s store.Store) *JsBundler {
	b := &JsBundler{}
	jsFile := pkgconfig.GetString("modules.bundle.js_file")
	if jsFile == "" {
		jsFile = "~/.config/cly/bundles/Jsfile"
	}
	mgr := strings.ToLower(strings.TrimSpace(pkgconfig.GetString("modules.bundle.js_manager")))
	if mgr == "" {
		mgr = "pnpm"
	}
	b.manager = mgr
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
	binPath := findWorkingManager(b.manager)
	if binPath == "" {
		return fmt.Errorf("%s not found or not working. Add to Brewfile: '%s', then run: cly bundle brew", b.manager, b.manager)
	}
	if b.manager == "pnpm" {
		// Ensure PNPM_HOME is set and in PATH for global installs
		pnpmHome := os.Getenv("PNPM_HOME")
		if pnpmHome == "" {
			home, _ := os.UserHomeDir()
			pnpmHome = home + "/Library/pnpm"
			os.Setenv("PNPM_HOME", pnpmHome)
		}
	}
	// Prepend the directory containing the working binary so all exec calls use it
	dir := binPath[:strings.LastIndex(binPath, "/")]
	path := os.Getenv("PATH")
	if !strings.HasPrefix(path, dir+":") {
		os.Setenv("PATH", dir+":"+path)
	}
	return nil
}

// findWorkingManager returns the path to a pnpm/npm binary that actually executes.
// It prefers the one in PATH but falls back to known locations if the PATH
// one is broken (e.g. ~/Library/pnpm/bin/pnpm pointing to a blank placeholder).
func findWorkingManager(name string) string {
	candidates := []string{}
	if p, err := exec.LookPath(name); err == nil {
		candidates = append(candidates, p)
	}
	// Known fallback locations
	candidates = append(candidates, "/opt/homebrew/bin/"+name, "/usr/local/bin/"+name)

	seen := map[string]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		cmd := exec.Command(p, "--version")
		if err := cmd.Run(); err == nil {
			return p
		}
	}
	return ""
}

// addSubcommand returns the install subcommand for the manager.
// pnpm uses "add", npm uses "install".
func (b *JsBundler) addSubcommand() string {
	if b.manager == "npm" {
		return "install"
	}
	return "add"
}

// removeSubcommand returns the uninstall subcommand for the manager.
func (b *JsBundler) removeSubcommand() string {
	if b.manager == "npm" {
		return "uninstall"
	}
	return "remove"
}

// jsFallbackRegistry returns the fallback registry configured in the
// global config; empty means no fallback and failed installs fail as-is.
func jsFallbackRegistry() string {
	return strings.TrimSpace(pkgconfig.GetString("modules.bundle.js_fallback_registry"))
}

// jsInstallWithFallback runs the manager install unchanged; when it fails
// and a fallback registry is configured, it retries once with
// --registry=<fallback>. The returned note is non-empty whenever the
// fallback was used, so callers can surface it to the user.
func jsInstallWithFallback(manager, addCmd, pkg string, force bool) (output, note string, err error) {
	output, err = runJsInstall(manager, addCmd, pkg, force, "")
	if err == nil {
		return output, "", nil
	}
	fallback := jsFallbackRegistry()
	if fallback == "" {
		return output, "", err
	}
	retryOutput, retryErr := runJsInstall(manager, addCmd, pkg, force, fallback)
	if retryErr == nil {
		note = fmt.Sprintf("%s failed on the default registry, installed via fallback registry %s", pkg, fallback)
		return retryOutput, note, nil
	}
	return output, "", fmt.Errorf("%w (fallback registry %s also failed: %v)", err, fallback, retryErr)
}

// runJsInstall runs `<manager> <addCmd> -g <pkg> [--registry=<url>] [--force]`.
func runJsInstall(manager, addCmd, pkg string, force bool, registry string) (string, error) {
	cmd := exec.Command(manager, jsInstallArgs(addCmd, pkg, force, registry)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s %s failed: %s", manager, addCmd, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func jsInstallArgs(addCmd, pkg string, force bool, registry string) []string {
	args := []string{addCmd, "-g", pkg}
	if registry != "" {
		args = append(args, "--registry="+registry)
	}
	if force {
		args = append(args, "--force")
	}
	return args
}

func (b *JsBundler) install(pkg string, verbose bool, force bool) error {
	_, note, err := jsInstallWithFallback(b.manager, b.addSubcommand(), pkg, force)
	if note != "" {
		printYellow(note)
	}
	return err
}

func (b *JsBundler) uninstall(pkg string, verbose bool) error {
	// Extract base package name (remove version spec)
	basePkg := extractJsBasePkg(pkg)
	cmd := exec.Command(b.manager, b.removeSubcommand(), "-g", basePkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s failed: %w", b.manager, b.removeSubcommand(), err)
	}
	return nil
}

// UninstallAll removes every package listed in the bundle file.
func (b *JsBundler) UninstallAll(bundleFile string, verbose bool) error {
	bundleFile = expandPath(bundleFile)
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}
	if len(desired) == 0 {
		printYellow("No packages listed in bundle file.")
		return nil
	}

	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	var toRemove []string
	var skipped []string
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		if _, ok := installed[base]; ok {
			toRemove = append(toRemove, base)
		} else {
			skipped = append(skipped, base)
		}
	}

	for _, base := range skipped {
		printYellow(fmt.Sprintf("Skipping (not installed): %s", base))
	}

	if len(toRemove) == 0 {
		printGreen("\nNothing to uninstall.")
		return nil
	}

	printGreen(fmt.Sprintf("Uninstalling %d packages with %s...", len(toRemove), b.manager))
	var failed int
	for _, base := range toRemove {
		printYellow(fmt.Sprintf("Removing: %s", base))
		if err := b.uninstall(base, verbose); err != nil {
			printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", base, err))
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d packages failed to uninstall", failed)
	}
	printGreen("\nDone!")
	return nil
}

// listInstalled returns map of package name -> version
func (b *JsBundler) listInstalled() (map[string]string, error) {
	switch b.manager {
	case "npm":
		return b.listInstalledNpm()
	case "bun":
		return b.listInstalledBun()
	}
	return b.listInstalledPnpm()
}

func (b *JsBundler) listInstalledPnpm() (map[string]string, error) {
	cmd := exec.Command("pnpm", "list", "-g", "--depth=0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list failed: %w", err)
	}

	packages := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	inDeps := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "dependencies:" {
			inDeps = true
			continue
		}
		if !inDeps || line == "" {
			continue
		}
		// Format: "@scope/name version" or "name version"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			packages[parts[0]] = ""
		}
	}
	return packages, nil
}

// listInstalledNpm parses `npm list -g --depth=0 --parseable=false` tree output.
// Lines look like: "├── pkg@1.0.0", "└── @scope/pkg@1.0.0".
func (b *JsBundler) listInstalledNpm() (map[string]string, error) {
	cmd := exec.Command("npm", "list", "-g", "--depth=0")
	output, _ := cmd.Output() // npm exits non-zero on extraneous deps; ignore err

	packages := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Strip leading tree characters and whitespace
		trimmed := strings.TrimLeft(line, " │├└─")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}
		// Skip the first line which is the global path (contains "/" but no leading tree char)
		if strings.HasPrefix(line, "/") || strings.HasPrefix(line, "~") {
			continue
		}
		// Skip annotations like "(empty)"
		if strings.HasPrefix(trimmed, "(") {
			continue
		}
		base := extractJsBasePkg(trimmed)
		version := ""
		if len(trimmed) > len(base)+1 {
			version = trimmed[len(base)+1:]
		}
		packages[base] = version
	}
	return packages, nil
}

// listInstalledBun parses `bun pm ls -g` tree output. Lines look like:
// "├── pkg@1.0.0", "└── @scope/pkg@1.0.0", with a header line naming the
// global node_modules path.
func (b *JsBundler) listInstalledBun() (map[string]string, error) {
	cmd := exec.Command("bun", "pm", "ls", "-g")
	output, err := cmd.Output()
	// bun exits non-zero on some odd states but still prints the tree; only
	// fail when it gave us nothing to parse. An empty global inventory
	// (fresh BUN_INSTALL) also prints nothing and exits non-zero.
	if err != nil && len(strings.TrimSpace(string(output))) == 0 {
		printYellow("bun pm ls -g returned nothing - treating global inventory as empty")
		return map[string]string{}, nil
	}

	return parseBunTree(string(output)), nil
}

// parseBunTree parses `bun pm ls -g` tree output into base -> version.
// Lines look like "├── pkg@1.0.0", "└── @scope/pkg@1.0.0"; the header line
// with the global node_modules path is skipped.
func parseBunTree(output string) map[string]string {
	packages := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(strings.TrimLeft(line, " │├└─"))
		if trimmed == "" || strings.HasPrefix(trimmed, "(") {
			continue
		}
		// Skip the header line with the global install path.
		if strings.HasPrefix(line, "/") || strings.HasPrefix(line, "~") {
			continue
		}
		base := extractJsBasePkg(trimmed)
		version := ""
		if len(trimmed) > len(base)+1 {
			version = trimmed[len(base)+1:]
		}
		// bun annotates outdated entries like "pkg@7.16.0 (v7.16.1
		// available)"; keep only the installed version.
		if i := strings.IndexByte(version, ' '); i >= 0 {
			version = version[:i]
		}
		packages[base] = version
	}
	return packages
}

// Sync installs/updates/removes packages to match Jsfile.
func (b *JsBundler) Sync(bundleFile string, verbose bool, force bool, noUpdate bool, taps bool, mas bool) error {
	bundleFile = expandPath(bundleFile)

	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	// Get installed packages (name -> version)
	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	// Build desired map (name -> version). Empty version = bare name, which
	// tracks the registry's latest dist-tag on every sync.
	desiredMap := jsDesiredMap(desired)

	// Remove packages not in desired
	for pkg := range installed {
		if _, want := desiredMap[pkg]; !want {
			printYellow(fmt.Sprintf("Removing: %s", pkg))
			if err := b.uninstall(pkg, verbose); err != nil {
				printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", pkg, err))
			}
		}
	}

	// Classify: installs for missing packages, upgrades/downgrades for
	// version moves, nothing for already-current entries.
	actions, examined, planErr := planJsSync(desired, installed, force, jsDefaultRegistry())
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
				return jsInstallWithFallback(b.manager, b.addSubcommand(), pkg, force)
			}
			if err := runParallelInstall(installFn, actions); err != nil {
				return err
			}
		} else {
			// Default: one action at a time so each is visible
			var failed int
			for _, action := range actions {
				printYellow(action.label())
				_, note, err := jsInstallWithFallback(b.manager, b.addSubcommand(), action.pkg, force)
				if note != "" {
					printYellow(note)
				}
				if err != nil {
					printRed(fmt.Sprintf("Warning: failed to sync %s: %v", action.base, err))
					failed++
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d packages failed to sync", failed)
			}
		}
	}

	printGreen("\n" + syncSummary(actions, examined))
	printGreen("Done!")
	return nil
}

// isLockedVersion reports true if the version spec starts with a digit or
// the standard semver prefixes used for exact/range locks (=, v, ~, ^).
// Floating dist-tags like "latest", "alpha", "beta", "next" return false
// and are always re-installed so they track upstream.

// Check reports drift between the bundle file and the real manager state
// using the same classification as Sync, including registry-backed checks
// for bare entries.
func (b *JsBundler) Check(bundleFile string) error {
	bundleFile = expandPath(bundleFile)
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}
	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}
	desiredMap := jsDesiredMap(desired)
	actions, examined, planErr := planJsSync(desired, installed, false, jsDefaultRegistry())
	if planErr != nil {
		printYellow("Warning: " + planErr.Error())
	}

	var extras []string
	for pkg := range installed {
		if _, want := desiredMap[pkg]; !want {
			extras = append(extras, pkg)
		}
	}
	sort.Strings(extras)

	if len(actions) == 0 && len(extras) == 0 {
		printGreen(fmt.Sprintf("Everything is in sync - %d packages up to date", examined))
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

// jsDesiredMap maps base package name -> version spec from the bundle.
// Empty version = bare name.
func jsDesiredMap(desired []string) map[string]string {
	m := make(map[string]string, len(desired))
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		version := ""
		if len(pkg) > len(base)+1 {
			version = pkg[len(base)+1:]
		}
		m[base] = version
	}
	return m
}

// planJsSync classifies every bundle entry against the installed inventory.
// Pinned entries compare locally; bare entries and floating dist-tags are
// resolved against registry (the only source for "latest"). Entries the
// registry cannot answer are left untouched and named in the returned
// error. Also returns the number of distinct bundle entries examined.
func planJsSync(desired []string, installed map[string]string, force bool, registry string) ([]syncAction, int, error) {
	desiredMap := jsDesiredMap(desired)

	// Entries with no local pin to compare against need an upstream lookup.
	tags := make(map[string]string) // base -> dist-tag to resolve
	for base, ver := range desiredMap {
		if ver == "" {
			tags[base] = "latest"
		} else if !isLockedVersion(ver) {
			tags[base] = ver
		}
	}
	var toResolve []string
	for base := range tags {
		if _, ok := installed[base]; ok {
			toResolve = append(toResolve, base)
		}
	}
	upstream := make(map[string]string)
	var planErr error
	if len(toResolve) > 0 {
		upstream, planErr = fetchLatestVersions(registry, toResolve, tags)
	}

	var actions []syncAction
	seen := make(map[string]bool, len(desired))
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		if seen[base] {
			continue
		}
		seen[base] = true
		desiredVer := normalizeVersion(desiredMap[base])
		installedVer, exists := installed[base]
		installedVer = normalizeVersion(installedVer)

		var verb, from, to string
		switch {
		case !exists:
			verb, to = "install", desiredVer
		case force:
			// --force reinstalls everything; version moves keep their labels.
			if desiredVer == "" || installedVer == desiredVer {
				verb = "reinstall"
			} else {
				verb, from, to = "upgrade", installedVer, desiredVer
				if compareVersions(installedVer, desiredVer) > 0 {
					verb = "downgrade"
				}
			}
		case desiredVer == "" || !isLockedVersion(desiredVer):
			// Bare name / floating tag tracks upstream; an unreachable
			// registry means the installed version stays untouched.
			up, ok := upstream[base]
			if !ok {
				continue
			}
			if up == installedVer {
				continue
			}
			verb, from, to = "upgrade", installedVer, up
			if compareVersions(installedVer, up) > 0 {
				verb = "downgrade"
			}
		case installedVer == desiredVer:
			continue
		default:
			// Pinned version moved - direction comes from the compare.
			verb, from, to = "upgrade", installedVer, desiredVer
			if compareVersions(installedVer, desiredVer) > 0 {
				verb = "downgrade"
			}
		}
		actions = append(actions, syncAction{pkg: pkg, base: base, verb: verb, from: from, to: to})
	}
	return actions, len(desiredMap), planErr
}

// jsDefaultRegistry resolves the registry the manager's plain installs go
// through: npm_config_registry env, then bunfig.toml / .npmrc in the home
// directory, finally the public npm registry.
func jsDefaultRegistry() string {
	if r := strings.TrimSpace(os.Getenv("npm_config_registry")); r != "" {
		return r
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, path := range []string{filepath.Join(home, ".bunfig.toml"), filepath.Join(home, ".npmrc")} {
			if r := registryFromNpmrcLike(path); r != "" {
				return r
			}
		}
	}
	return "https://registry.npmjs.org/"
}

// registryFromNpmrcLike scans a bunfig.toml/.npmrc file for a top-level
// `registry = value` entry.
func registryFromNpmrcLike(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// bunfig.toml writes `install.registry = value`; .npmrc writes
		// `registry=value`. Both reduce to the same key.
		line = strings.TrimPrefix(line, "install.")
		if !strings.HasPrefix(line, "registry") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "registry"))
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		rest = strings.Trim(strings.TrimSpace(rest[1:]), `"'`)
		if rest != "" {
			return rest
		}
	}
	return ""
}

// fetchLatestVersions resolves the given dist-tag for each base package
// concurrently (metadata requests only, no tarballs). Resolved entries are
// returned; failures are named in the error and simply absent from the map
// so callers can leave those packages untouched.
func fetchLatestVersions(registry string, bases []string, tags map[string]string) (map[string]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return fetchConcurrent(bases, tags, func(base string) (string, error) {
		return fetchDistTag(client, registry, base, tags[base])
	})
}

// fetchDistTag fetches {registry}/{pkg}/{tag} and returns its version.
func fetchDistTag(client *http.Client, registry, pkg, tag string) (string, error) {
	url := strings.TrimSuffix(registry, "/") + "/" + strings.Replace(pkg, "/", "%2F", 1) + "/" + tag
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %d", url, resp.StatusCode)
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return "", err
	}
	if manifest.Version == "" {
		return "", fmt.Errorf("%s returned no version", url)
	}
	return manifest.Version, nil
}

func isLockedVersion(v string) bool {
	if v == "" {
		return false
	}
	c := v[0]
	return (c >= '0' && c <= '9') || c == '=' || c == 'v' || c == '~' || c == '^'
}

// extractJsBasePkg extracts base package name from spec.
// @scope/pkg@1.0.0 → @scope/pkg
// pkg@1.0.0 → pkg
func extractJsBasePkg(pkg string) string {
	// Handle scoped packages (@scope/name@version)
	if strings.HasPrefix(pkg, "@") {
		// Find the second @ which is the version separator
		rest := pkg[1:]
		if idx := strings.Index(rest, "@"); idx > 0 {
			return pkg[:idx+1]
		}
		return pkg
	}
	// Regular package
	if idx := strings.Index(pkg, "@"); idx > 0 {
		return pkg[:idx]
	}
	return pkg
}
