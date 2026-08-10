package dotfiles

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/cmux"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/mut"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	installFlag     bool
	opFlag          bool
	allFlag         bool
	cacheFlag       bool
	installOnlyFlag    bool
	installNoAIFlag    bool
	reinstallFlag      bool
	forceFlag       bool
	verboseFlag     bool
	dryRunFlag      bool
	failFastFlag    bool
	bypassAIFlag    bool
	configFlag      string
	noItFlag        bool
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "dotfiles",
		Short: "Manage dotfile symlinks",
		Long: `Create and manage symlinks from a declarative config file.

Config syntax (dotfiles.conf):
  ./src -> ~/dst                        symlink (dirs need trailing /)
  !cmd                                  run shell command (-i flag)
  @install <url>                        fetch+analyze install script (-i flag)
  @cache <command>                      runs once; re-runs only when the command text changes (sha256-keyed)
  @op account=x ./s.op -> ~/d           1Password inject (-o flag)
  @op account=x op://vault/item/field -> ~/d  1Password read secret (-o flag)
  .jsonc -> .json                       comments stripped automatically
  @target user=u,.. os=linux,darwin arch=amd64,..   gate: skip unless this machine matches

Config discovery: candidates are tried in order — dotfiles.<user>.conf then
dotfiles.conf — and the first whose @target header matches this machine's
user/os/arch is loaded; the search stops there. Exactly one file is loaded, and
if none match (e.g. root, or the wrong OS) nothing is applied.

Use --cache to force re-run of every @cache entry (ignores the hash skip).
Use --install-only to run only @install directives (skips everything else).
Use --install-no-ai to run only @install directives, skipping LLM analysis.

Maintenance:
  cly dotfiles prune                    dry-run cleanup of stale cache entries
  cly dotfiles prune --apply            actually drop entries no longer in the conf`,
		RunE:  runSync,
	}

	cmd.Flags().BoolVarP(&installFlag, "install", "i", false, "Execute install commands (lines starting with !)")
	cmd.Flags().BoolVarP(&opFlag, "op", "o", false, "Inject 1Password templates")
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Run everything (install, 1Password)")
	cmd.Flags().BoolVar(&cacheFlag, "cache", false, "Run all @cache entries (bypass the 'already installed' check; during normal sync the check is honored)")
	cmd.PersistentFlags().BoolVarP(&forceFlag, "force", "f", false, "Force actions")
	cmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "Verbose output (show overwrites, intermediate steps)")
	cmd.PersistentFlags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Dry run: log every mutation (fs writes, shell commands) without executing")
	cmd.PersistentFlags().BoolVar(&failFastFlag, "fail-fast", false, "Abort sync on the first error (default: print the error and continue)")
	cmd.PersistentPreRun = func(c *cobra.Command, args []string) { mut.SetDryRun(dryRunFlag) }
	cmd.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "Path to config file (default: <dotfiles_dir>/dotfiles.conf)")
	cmd.Flags().BoolVar(&noItFlag, "no-it", false, "Skip interactive prompts (non-interactive mode)")
	cmd.Flags().BoolVar(&bypassAIFlag, "bypass-ai", false, "Skip LLM analysis for @install directives (no uninstall manifest)")
	cmd.Flags().BoolVar(&installOnlyFlag, "install-only", false, "Run only @install directives (skips symlinks, jobs, op)")
	cmd.Flags().BoolVar(&installNoAIFlag, "install-no-ai", false, "Run only @install directives, skip LLM analysis")
	cmd.Flags().BoolVar(&reinstallFlag, "reinstall", false, "Force reinstall @install directives even if SHA unchanged")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all mappings",
		RunE:  runStatus,
	}

	unlinkCmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove all managed symlinks",
		RunE:  runUnlink,
	}

	evalCmd := &cobra.Command{
		Use:   "eval [src]",
		Short: "Apply a single mapping from 'source -> destination' format",
		Long: `Apply a single dotfiles mapping directly.

Example:
  cly dotfiles eval "./home/.pi/agent/settings.jsonc -> ~/.pi/agent/settings.json"
  echo "./home/.pi/agent/settings.jsonc -> ~/.pi/agent/settings.json" | cly dotfiles eval`,
		RunE: runEval,
	}

	pruneCmd := &cobra.Command{
		Use:   "prune",
		Short: "Drop stale cache lock entries (dry-run unless --apply)",
		RunE:  runPrune,
	}
	pruneCmd.Flags().Bool("apply", false, "Actually remove flagged entries (default: dry-run)")
	pruneCmd.Flags().Duration("max-age", cacheGracePeriod, "Grace window before flagged entries are eligible for pruning")

	cmd.AddCommand(statusCmd, unlinkCmd, evalCmd, pruneCmd)
	parent.AddCommand(cmd)
}

func runEval(cmd *cobra.Command, args []string) error {
	var input string
	if len(args) > 0 {
		input = strings.TrimSpace(strings.Join(args, " "))
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := os.ReadFile("/dev/stdin")
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			input = strings.TrimSpace(string(data))
		} else {
			return fmt.Errorf("provide a mapping (e.g. './src -> ~/dest') as argument or via stdin")
		}
	}

	// Parse "source -> dest" mapping format
	m, err := ParseMapping(input)
	if err != nil {
		return fmt.Errorf("invalid mapping: %w", err)
	}

	if IsJsoncToJson(m) {
		result := CopyJsoncToJson(m)
		printJsoncResult(m, result)
		return nil
	}
	result := CreateSymlink(m)
	printResult(m, result)
	return nil
}

// getConfigPath selects the single dotfiles config for this machine. It walks
// candidates in priority order (dotfiles.<user>.conf, then dotfiles.conf) and
// returns the first whose @target header matches the current user/os/arch. The
// header is the selector: the first match wins and the search stops. If no
// candidate matches (e.g. root, or the wrong OS), it errors instead of applying
// an unintended config.
func getConfigPath() (string, error) {
	if configFlag != "" {
		return configFlag, nil
	}

	cfg := pkgconfig.Get()
	dotfilesDir := "~/DotFiles"
	if cfg != nil && cfg.App.DotFilesDir != "" {
		dotfilesDir = cfg.App.DotFilesDir
	}
	dotfilesDir = expandTilde(dotfilesDir)

	var candidates []string
	if u := currentUsername(); u != "" {
		candidates = append(candidates, filepath.Join(dotfilesDir, fmt.Sprintf("dotfiles.%s.conf", u)))
	}
	candidates = append(candidates, filepath.Join(dotfilesDir, "dotfiles.conf"))

	var rejected []string
	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		parsed, err := ParseConfig(p)
		if err != nil {
			return "", err
		}
		if reason := parsed.Target.GateReason(); reason == "" {
			return p, nil
		} else {
			rejected = append(rejected, fmt.Sprintf("%s: %s", filepath.Base(p), reason))
		}
	}

	if len(rejected) > 0 {
		return "", fmt.Errorf("no dotfiles config matches this machine:\n  %s", strings.Join(rejected, "\n  "))
	}
	return "", fmt.Errorf("no dotfiles config found in %s (looked for dotfiles.<user>.conf, dotfiles.conf)", dotfilesDir)
}

func runSync(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s\nCreate it or use --config /path/to/dotfiles.conf", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	// Load previous lock before applying anything.
	lockFile, _ := lockFilePath()
	oldLock, _ := loadLock(lockFile)

	for _, e := range cfg.Errors {
		fmt.Printf("⚠️  %s\n", e)
	}

	// --cache short-circuits the rest of sync: re-run every @cache entry
	// regardless of the hash skip. The lock is still updated so subsequent
	// runs benefit from the refreshed metadata.
	if cacheFlag {
		cacheLock := buildLock(cfg, oldLock)
		if err := ApplyCache(cfg, cacheLock, CacheApplyOptions{Force: true, FailFast: failFastFlag}); err != nil {
			return err
		}
		pruneStaleCacheEntries(cacheLock, cfg, time.Now().UTC(), false)
		return saveLock(lockFile, cacheLock)
	}
	if installOnlyFlag || installNoAIFlag {
		return runInstallsOnly(cfg, installNoAIFlag || bypassAIFlag)
	}

	if !mut.DryRun() && !noItFlag {
		if err := confirmBackups(cfg, oldLock); err != nil {
			return err
		}
	}

	for _, m := range cfg.Mappings {
		var result LinkResult
		if IsJsoncToJson(m) {
			result = CopyJsoncToJson(m)
			printJsoncResult(m, result)
		} else {
			result = CreateSymlink(m)
			printResult(m, result)
		}
		if failFastFlag && (result.State == StateError || result.State == StateConflict) {
			return fmt.Errorf("%s: %s", m.Destination, result.Error)
		}
	}

	if len(cfg.InstallCommands) > 0 || len(cfg.Installs) > 0 {
		if installFlag || allFlag {
			for _, cmdStr := range cfg.InstallCommands {
				fmt.Printf("%s %s\n", style.BlueStyle.Render("⚡ Executing:"), cmdStr)
				if err := executeCommand(cmdStr, cfg.BaseDir); err != nil {
					fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
					if failFastFlag {
						return fmt.Errorf("install command failed: %w", err)
					}
				}
			}
			if len(cfg.Installs) > 0 {
				fmt.Printf("\n%s Applying %d @install directive(s)\n", style.BlueStyle.Render("⚙️"), len(cfg.Installs))
				if err := ApplyInstalls(cfg, InstallOptions{
					BypassAI:  bypassAIFlag,
					Reinstall: reinstallFlag,
					FailFast:  failFastFlag,
				}); err != nil {
					return err
				}
			}
		} else {
			skipped := len(cfg.InstallCommands) + len(cfg.Installs)
			fmt.Printf("\n%s %d install command(s)/@install directive(s) skipped (use -i to execute)\n",
				style.YellowStyle.Render("⏭️ "), skipped)
		}
	}

	if len(cfg.OpMappings) > 0 {
		if opFlag || allFlag {
			fmt.Printf("\n%s Injecting %d 1Password template(s)\n", style.BlueStyle.Render("🔑"), len(cfg.OpMappings))
			if err := ApplyOpMappings(cfg, failFastFlag); err != nil {
				return err
			}
		} else {
			fmt.Printf("\n%s %d 1Password template(s) skipped (use -o to inject)\n",
				style.YellowStyle.Render("⏭️ "), len(cfg.OpMappings))
		}
	}

	// Reload lock to pick up hashes written by ApplyInstalls.
	postApplyLock, _ := loadLock(lockFile)

	// Build new lock (carries forward cache metadata + stale candidates).
	newLock := buildLock(cfg, postApplyLock)

	if len(cfg.CacheEntries) > 0 {
		fmt.Printf("\n%s Applying %d @cache entry/entries\n", style.BlueStyle.Render("⚙️"), len(cfg.CacheEntries))
		if err := ApplyCache(cfg, newLock, CacheApplyOptions{Force: forceFlag, FailFast: failFastFlag}); err != nil {
			return err
		}
	}

	// Auto-prune: flag stale entries, drop those past the grace window.
	pruneStaleCacheEntries(newLock, cfg, time.Now().UTC(), false)

	diff := diffLocks(oldLock, newLock)
	applyDiff(diff)

	// Save updated lock.
	_ = saveLock(lockFile, newLock)

	cmux.Notify(cmd.Context(), "Dotfiles", "Sync complete")
	return nil
}

// applyDiff cleans up artifacts that were removed from dotfiles.conf since the last run.
func applyDiff(diff LockDiff) {
	for _, e := range diff.RemovedSymlinks {
		m := Mapping{Source: e.Source, Destination: e.Destination}
		if RemoveSymlink(m) {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed symlink:"), shortenPath(e.Destination))
		}
	}

	for _, e := range diff.RemovedJsoncCopies {
		m := Mapping{Source: e.Source, Destination: e.Destination}
		if RemoveJsoncCopy(m) {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed jsonc copy:"), shortenPath(e.Destination))
		}
	}

	for _, e := range diff.RemovedInstalls {
		RemoveInstallArtifacts(e)
	}

	for _, e := range diff.RemovedOpMappings {
		m := OpMapping{Source: e.Source, Destination: e.Destination}
		if RemoveOpMapping(m) {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed op file:"), shortenPath(e.Destination))
		}
	}

	if len(diff.RemovedInstallCommands) > 0 && !noItFlag {
		fmt.Printf("\n%s\n", style.RedStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Printf("%s\n", style.RedStyle.Render("  ⛔  REMOVED INSTALL COMMANDS — manual cleanup required"))
		fmt.Printf("%s\n", style.RedStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		for _, cmdStr := range diff.RemovedInstallCommands {
			fmt.Printf("  %s  %s\n", style.RedStyle.Render("▶"), cmdStr)
		}
		fmt.Printf("%s\n\n", style.RedStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Printf("These commands were previously executed and cannot be auto-undone.\n")
		fmt.Printf("Review the list above and undo manually if needed.\n\n")
		fmt.Printf("Press Enter to continue...")
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Dotfiles: %s\n\n", configPath)

	for _, m := range cfg.Mappings {
		result := CheckStatus(m)
		printStatusResult(m, result)
	}

	if len(cfg.InstallCommands) > 0 {
		fmt.Printf("\nInstall commands: %d (use -i to execute)\n", len(cfg.InstallCommands))
	}
	if len(cfg.CacheEntries) > 0 {
		fmt.Printf("Cache entries: %d\n", len(cfg.CacheEntries))
	}
	if len(cfg.OpMappings) > 0 {
		fmt.Printf("1Password templates: %d (use -o to inject)\n", len(cfg.OpMappings))
	}

	return nil
}

func runUnlink(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	removed := 0
	for _, m := range cfg.Mappings {
		if RemoveSymlink(m) {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed:"), shortenPath(m.Destination))
			removed++
		}
	}

	fmt.Printf("\n%s Removed %d symlink(s)\n", style.GreenStyle.Render("✅"), removed)
	return nil
}

func printResult(m Mapping, r LinkResult) {
	src := shortenPath(m.Source)
	dest := shortenPath(m.Destination)

	if verboseFlag {
		fmt.Printf("%s %s -> %s\n",
			style.BlueStyle.Render("🔗 Processing:"),
			src, dest)
	}

	switch r.State {
	case StateLinked:
		if r.CreatedDir && verboseFlag {
			fmt.Printf("  %s %s\n",
				style.BlueStyle.Render("📁 Creating directory:"),
				filepath.Dir(dest))
		}
		if r.RemovedExisting && verboseFlag {
			if r.BackupPath != "" {
				fmt.Printf("  %s %s -> %s\n",
					style.YellowStyle.Render("📦 Backed up:"),
					dest, shortenPath(r.BackupPath))
			} else {
				fmt.Printf("  %s %s\n",
					style.YellowStyle.Render("🗑️  Removing existing:"),
					dest)
			}
		}
		fmt.Printf("%s %s -> %s\n",
			style.GreenStyle.Render("✅ Symlink:"),
			src, dest)
	case StateMissing:
		fmt.Printf("  %s Source '%s' does not exist, skipping\n",
			style.YellowStyle.Render("⚠️  Warning:"),
			src)
	case StateConflict:
		fmt.Printf("  %s %s\n",
			style.RedStyle.Render("❌ Error:"),
			r.Error)
	case StateError:
		fmt.Printf("  %s %s\n",
			style.RedStyle.Render("❌ Error:"),
			r.Error)
	}
}

func printJsoncResult(m Mapping, r LinkResult) {
	src := shortenPath(m.Source)
	dest := shortenPath(m.Destination)

	if verboseFlag {
		fmt.Printf("%s %s -> %s\n",
			style.BlueStyle.Render("📄 Copying (jsonc→json):"),
			src, dest)
	}

	switch r.State {
	case StateLinked:
		if r.CreatedDir && verboseFlag {
			fmt.Printf("  %s %s\n",
				style.BlueStyle.Render("📁 Creating directory:"),
				filepath.Dir(dest))
		}
		if r.RemovedExisting && verboseFlag {
			if r.BackupPath != "" {
				fmt.Printf("  %s %s -> %s\n",
					style.YellowStyle.Render("📦 Backed up:"),
					dest, shortenPath(r.BackupPath))
			} else {
				fmt.Printf("  %s %s\n",
					style.YellowStyle.Render("🗑️  Overwriting:"),
					dest)
			}
		}
		fmt.Printf("%s %s -> %s\n",
			style.GreenStyle.Render("✅ Copied (jsonc→json):"),
			src, dest)
	case StateMissing:
		fmt.Printf("  %s Source '%s' does not exist, skipping\n",
			style.YellowStyle.Render("⚠️  Warning:"),
			src)
	case StateError:
		fmt.Printf("  %s %s\n",
			style.RedStyle.Render("❌ Error:"),
			r.Error)
	}
}

func printStatusResult(m Mapping, r LinkResult) {
	dest := shortenPath(m.Destination)
	src := shortenPath(m.Source)
	switch r.State {
	case StateLinked:
		fmt.Printf("%s %-40s → %s\n", style.GreenStyle.Render("✓"), dest, src)
	case StateUnlinked:
		fmt.Printf("%s %-40s → %s\n", style.SubtleStyle.Render("○"), dest, src)
	case StateMissing:
		fmt.Printf("%s %-40s (source missing)\n", style.YellowStyle.Render("⚠️ "), dest)
	case StateConflict:
		fmt.Printf("%s %-40s (conflict)\n", style.RedStyle.Render("✗"), dest)
	case StateBroken:
		fmt.Printf("%s %-40s (broken symlink)\n", style.RedStyle.Render("✗"), dest)
	}
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func runInstallsOnly(cfg *Config, bypassAI bool) error {
	if len(cfg.Installs) == 0 {
		fmt.Println("No @install directives declared.")
		return nil
	}
	fmt.Printf("%s Applying %d @install directive(s)\n", style.BlueStyle.Render("⚙️"), len(cfg.Installs))
	return ApplyInstalls(cfg, InstallOptions{BypassAI: bypassAI, Reinstall: reinstallFlag, FailFast: failFastFlag})
}

func executeCommand(cmdStr, baseDir string) error {
	if strings.HasPrefix(cmdStr, "zellij_plugin ") {
		url := strings.TrimPrefix(cmdStr, "zellij_plugin ")
		url = strings.TrimSpace(url)
		if mut.DryRun() {
			fmt.Printf("%s zellij-plugin %s\n", style.YellowStyle.Render("[dry-run]"), url)
			return nil
		}
		return downloadZellijPlugin(url)
	}
	return mut.ExecDir(baseDir, "fish", "-c", cmdStr)
}


// runPrune implements `cly dotfiles prune`. By default it is a dry run that
// reports stale @cache lock entries; --apply commits the deletions. The
// orphan symlink/jsonc/op/install paths are already handled by the normal
// sync diff/apply machinery (see applyDiff), so prune intentionally only
// touches the cache section.
func runPrune(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	apply, _ := cmd.Flags().GetBool("apply")
	maxAge, _ := cmd.Flags().GetDuration("max-age")
	if maxAge <= 0 {
		maxAge = cacheGracePeriod
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	lockFile, _ := lockFilePath()
	lock, _ := loadLock(lockFile)

	cfgHashes := make(map[string]bool, len(cfg.CacheEntries))
	for _, e := range cfg.CacheEntries {
		cfgHashes[hashCacheEntry(e)] = true
	}

	now := time.Now().UTC()
	var pastGrace, inGrace []CacheLockEntry
	for _, e := range lock.Cache {
		if cfgHashes[e.Hash] {
			continue
		}
		if e.FlaggedForDelete == "" {
			// Never flagged yet — would be flagged on next sync.
			inGrace = append(inGrace, e)
			continue
		}
		flaggedAt, perr := time.Parse(time.RFC3339, e.FlaggedForDelete)
		if perr != nil || now.Sub(flaggedAt) >= maxAge {
			pastGrace = append(pastGrace, e)
		} else {
			inGrace = append(inGrace, e)
		}
	}

	fmt.Printf("%s Scanning %s\n", style.BlueStyle.Render("🔍"), shortenPath(configPath))
	fmt.Printf("  cache entries: %d in config, %d in lock\n", len(cfg.CacheEntries), len(lock.Cache))
	fmt.Printf("  flagged for prune (>= %s): %d\n", maxAge, len(pastGrace))
	for _, e := range pastGrace {
		fmt.Printf("    - %s: %s (flagged %s, last run %s, exit %d)\n", shortHash(e.Hash), truncateCmd(e.Command), fmtRFC(e.FlaggedForDelete), fmtRFC(e.LastRun), e.ExitCode)
	}
	fmt.Printf("  flagged but in grace window: %d\n", len(inGrace))
	for _, e := range inGrace {
		ago := ""
		if t, err := time.Parse(time.RFC3339, e.FlaggedForDelete); err == nil {
			ago = fmt.Sprintf(", %s ago", now.Sub(t).Truncate(time.Second))
		}
		fmt.Printf("    - %s: %s (flagged %s%s)\n", shortHash(e.Hash), truncateCmd(e.Command), fmtRFC(e.FlaggedForDelete), ago)
	}

	if !apply {
		count := len(pastGrace)
		fmt.Printf("%s Would prune %d cache entry/entries. Run with --apply to commit.\n", style.YellowStyle.Render("[dry-run]"), count)
		return nil
	}

	// Hard prune: drop every stale entry regardless of age.
	_, _, pruned := pruneStaleCacheEntries(lock, cfg, now, true)
	if err := saveLock(lockFile, lock); err != nil {
		return err
	}
	fmt.Printf("%s Pruned %d cache entry/entries.\n", style.GreenStyle.Render("✅"), len(pruned))
	return nil
}

func fmtRFC(s string) string {
	if s == "" {
		return "—"
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02")
	}
	return s
}
