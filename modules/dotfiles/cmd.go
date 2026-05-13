package dotfiles

import (
	"github.com/yurifrl/cly/pkg/mut"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/cmux"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	installFlag bool
	jobsFlag    bool
	opFlag      bool
	allFlag     bool
	onceFlag    bool
	cacheFlag   bool
	forceFlag   bool
	verboseFlag bool
	dryRunFlag  bool
	configFlag  string
	noItFlag    bool
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "dotfiles",
		Short: "Manage dotfile symlinks",
		Long: `Create and manage symlinks from a declarative config file.

Config syntax (dotfiles.conf):
  ./src -> ~/dst                        symlink (dirs need trailing /)
  !cmd                                  run shell command (-i flag)
  @once name -- cmd                     run once ever (-f to rerun)
  @cache name -- cmd                    run unless 'command -v name' exists
  @startup name -- cmd                  run every time (-j flag)
  @startup name keepalive -- cmd        run and keep alive
  @interval name every=1h -- cmd        run if interval elapsed
  @op account=x ./s.op -> ~/d           1Password inject (-o flag)
  @op account=x op://vault/item/field -> ~/d  1Password read secret (-o flag)
  .jsonc -> .json                       comments stripped automatically

Use --once to run only @once jobs (skips everything else).
Use --cache to run only @cache jobs (skips everything else).`,
		RunE:  runSync,
	}

	cmd.Flags().BoolVarP(&installFlag, "install", "i", false, "Execute install commands (lines starting with !)")
	cmd.Flags().BoolVarP(&jobsFlag, "jobs", "j", false, "Apply declarative jobs (@startup/@interval/@once)")
	cmd.Flags().BoolVarP(&opFlag, "op", "o", false, "Inject 1Password templates")
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Run everything (install, jobs, 1Password)")
	cmd.Flags().BoolVar(&onceFlag, "once", false, "Run only @once jobs (skips symlinks, install, op, startup, interval)")
	cmd.Flags().BoolVar(&cacheFlag, "cache", false, "Run all @cache jobs (bypass the 'already installed' check; during normal sync the check is honored)")
	cmd.PersistentFlags().BoolVarP(&forceFlag, "force", "f", false, "Force actions (rerun @once jobs)")
	cmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "Verbose output (show overwrites, intermediate steps)")
	cmd.PersistentFlags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Dry run: log every mutation (fs writes, shell commands) without executing")
	cmd.PersistentPreRun = func(c *cobra.Command, args []string) { mut.SetDryRun(dryRunFlag) }
	cmd.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "Path to config file (default: <dotfiles_dir>/dotfiles.conf)")
	cmd.Flags().BoolVar(&noItFlag, "no-it", false, "Skip interactive prompts (non-interactive mode)")

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

	cmd.AddCommand(statusCmd, unlinkCmd, evalCmd)
	registerJobsCommands(cmd)
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

	return filepath.Join(dotfilesDir, "dotfiles.conf"), nil
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

	// --once short-circuits everything else: only @once jobs are applied.
	if onceFlag {
		return runJobsSubset(cmd, cfg, JobRunOnce, "@once", forceFlag)
	}
	if cacheFlag {
		// Explicit --cache means the user wants to (re)run cache jobs regardless
		// of whether the binary is already on PATH. During a normal sync (-j/-a)
		// the cache check is honored.
		return runJobsSubset(cmd, cfg, JobRunCache, "@cache", true)
	}

	for _, m := range cfg.Mappings {
		if IsJsoncToJson(m) {
			result := CopyJsoncToJson(m)
			printJsoncResult(m, result)
		} else {
			result := CreateSymlink(m)
			printResult(m, result)
		}
	}

	if len(cfg.InstallCommands) > 0 {
		if installFlag || allFlag {
			for _, cmdStr := range cfg.InstallCommands {
				fmt.Printf("%s %s\n", style.BlueStyle.Render("⚡ Executing:"), cmdStr)
				if err := executeCommand(cmdStr, cfg.BaseDir); err != nil {
					fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
				}
			}
		} else {
			fmt.Printf("\n%s %d install command(s) skipped (use -i to execute)\n",
				style.YellowStyle.Render("⏭️ "), len(cfg.InstallCommands))
		}
	}

	if len(cfg.OpMappings) > 0 {
		if opFlag || allFlag {
			fmt.Printf("\n%s Injecting %d 1Password template(s)\n", style.BlueStyle.Render("🔑"), len(cfg.OpMappings))
			if err := ApplyOpMappings(cfg); err != nil {
				return err
			}
		} else {
			fmt.Printf("\n%s %d 1Password template(s) skipped (use -o to inject)\n",
				style.YellowStyle.Render("⏭️ "), len(cfg.OpMappings))
		}
	}

	if len(cfg.Jobs) > 0 {
		if jobsFlag || allFlag {
			fmt.Printf("\n%s Applying %d job(s)\n", style.BlueStyle.Render("⚙️"), len(cfg.Jobs))
			if err := ApplyJobs(cfg, JobApplyOptions{Force: forceFlag}); err != nil {
				return err
			}
		} else {
			fmt.Printf("\n%s %d job(s) skipped (use -j to apply)\n",
				style.YellowStyle.Render("⏭️ "), len(cfg.Jobs))
		}
	}

	// Build new lock and diff against previous.
	newLock := buildLock(cfg)
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

	for _, name := range diff.RemovedJobs {
		if err := RemoveJobByName(name); err == nil {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed job:"), name)
		}
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
	if len(cfg.Jobs) > 0 {
		fmt.Printf("Jobs: %d (use 'dotfiles jobs status' for details)\n", len(cfg.Jobs))
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
			fmt.Printf("  %s %s\n",
				style.YellowStyle.Render("🗑️  Removing existing:"),
				dest)
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
			fmt.Printf("  %s %s\n",
				style.YellowStyle.Render("🗑️  Overwriting:"),
				dest)
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

func runJobsSubset(cmd *cobra.Command, cfg *Config, run JobRun, label string, force bool) error {
	filtered := make([]Job, 0, len(cfg.Jobs))
	for _, j := range cfg.Jobs {
		if j.Run == run {
			filtered = append(filtered, j)
		}
	}
	if len(filtered) == 0 {
		fmt.Printf("No %s jobs declared.\n", label)
		return nil
	}
	subset := *cfg
	subset.Jobs = filtered
	fmt.Printf("%s Applying %d %s job(s)\n", style.BlueStyle.Render("⚙️"), len(filtered), label)
	if err := ApplyJobs(&subset, JobApplyOptions{Force: force}); err != nil {
		return err
	}
	cmux.Notify(cmd.Context(), "Dotfiles", fmt.Sprintf("Ran %d %s job(s)", len(filtered), label))
	return nil
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
