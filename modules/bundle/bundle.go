package bundle

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/store"
)

var (
	fileFlag      string
	verboseFlag   bool
	noItFlag      bool
	forceFlag     bool
	upgradeFlag   bool
	noUpdateFlag  bool
	tapsFlag      bool
	noCleanupFlag bool
	masFlag       bool
	parallelFlag  bool
	updateFlag    bool
	uninstallFlag bool
)

// Register adds the bundle command and subcommands to the root command.
func Register(root *cobra.Command) {
	getBundlers := func() (map[string]Bundler, func(), error) {
		s, err := openStore()
		if err != nil {
			return nil, nil, err
		}
		bundlers := map[string]Bundler{
			"brew":   NewBrewBundler(),
			"js":     NewJsBundler(s),
			"python": NewPythonBundler(s),
			"py":     NewPythonBundler(s),
			"rust":   NewRustBundler(s),
		}
		cleanup := func() {
			if err := s.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close store: %v\n", err)
			}
		}
		return bundlers, cleanup, nil
	}

	cmd := &cobra.Command{
		Use:          "bundle [type]",
		Short:        "Edit and sync packages from declarative bundle files",
		SilenceUsage: true,
		Long: `Unified declarative package management for brew, js, python, and rust.

Opens bundle file in $EDITOR, syncs after save, prompts to continue editing.

Types:
  brew       Sync Homebrew packages from Brewfile (default)
  js         Sync JavaScript packages from package.json
  python/py  Sync Python tools from Pythonfile
  rust       Sync Rust crates from Rsfile`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundlers, cleanup, err := getBundlers()
			if err != nil {
				return fmt.Errorf("failed to initialize bundlers: %w", err)
			}
			defer cleanup()

			bundleType := "brew"
			if len(args) > 0 {
				bundleType = args[0]
			}
			if noItFlag || updateFlag || uninstallFlag {
				return runSync(bundlers, bundleType)
			}
			return runIterative(bundlers, bundleType)
		},
	}

	cmd.Flags().StringVarP(&fileFlag, "file", "f", "", "override bundle file path")
	cmd.Flags().BoolVar(&verboseFlag, "verbose", false, "show detailed output")
	cmd.Flags().BoolVar(&noItFlag, "no-it", false, "skip interactive editor mode, just sync")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "force reinstall packages even if already installed")
	cmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "force upgrade packages to latest (js only)")
	cmd.Flags().BoolVar(&noUpdateFlag, "no-update", false, "skip brew upgrade (brew only)")
	cmd.Flags().BoolVar(&tapsFlag, "taps", false, "install taps first (brew only)")
	cmd.Flags().BoolVar(&noCleanupFlag, "no-cleanup", false, "skip cleanup after sync")
	cmd.Flags().BoolVar(&masFlag, "mas", false, "install Mac App Store apps (brew only)")
	cmd.Flags().BoolVar(&parallelFlag, "parallel", false, "use parallel installs with TUI progress (js only)")
	cmd.Flags().BoolVarP(&updateFlag, "update", "u", false, "sync without opening the editor")
	cmd.Flags().BoolVar(&uninstallFlag, "uninstall", false, "uninstall all packages listed in the bundle file (js only)")

	cmd.AddCommand(checkCmd(getBundlers))
	cmd.AddCommand(cleanupCmd(getBundlers))

	root.AddCommand(cmd)
}

func openStore() (store.Store, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = home + "/.local/share/cly"
	} else {
		dataDir = dataDir + "/cly"
	}

	dbPath := dataDir + "/cly.db"
	return store.New(dbPath)
}

func checkCmd(getBundlers func() (map[string]Bundler, func(), error)) *cobra.Command {
	return &cobra.Command{
		Use:   "check [type]",
		Short: "Show what would change without making changes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundlers, cleanup, err := getBundlers()
			if err != nil {
				return fmt.Errorf("failed to initialize bundlers: %w", err)
			}
			defer cleanup()

			bundleType := "brew"
			if len(args) > 0 {
				bundleType = args[0]
			}
			return runCheck(bundlers, bundleType)
		},
	}
}

func cleanupCmd(getBundlers func() (map[string]Bundler, func(), error)) *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup [type]",
		Short: "Remove packages not in bundle file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundlers, cleanup, err := getBundlers()
			if err != nil {
				return fmt.Errorf("failed to initialize bundlers: %w", err)
			}
			defer cleanup()

			bundleType := "brew"
			if len(args) > 0 {
				bundleType = args[0]
			}
			return runCleanup(bundlers, bundleType)
		},
	}
}

func runSync(bundlers map[string]Bundler, bundleType string) error {
	if uninstallFlag {
		if bundleType != "js" {
			return fmt.Errorf("--uninstall is only supported for 'js' bundle type")
		}
		jsB, ok := bundlers["js"].(*JsBundler)
		if !ok {
			return fmt.Errorf("js bundler not available")
		}
		if err := jsB.CheckDeps(); err != nil {
			return err
		}
		return jsB.UninstallAll(getBundleFile(jsB), verboseFlag)
	}

	if bundleType == "all" {
		return runAll(bundlers, func(b Bundler) error {
			return b.Sync(getBundleFile(b), verboseFlag, forceFlag, noUpdateFlag, tapsFlag, masFlag)
		})
	}

	bundler, ok := bundlers[bundleType]
	if !ok {
		return fmt.Errorf("unknown bundle type: %s (valid: brew, js, python/py, rust, all)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	return bundler.Sync(getBundleFile(bundler), verboseFlag, forceFlag, noUpdateFlag, tapsFlag, masFlag)
}

func runCheck(bundlers map[string]Bundler, bundleType string) error {
	if bundleType == "all" {
		return runAll(bundlers, func(b Bundler) error {
			return b.Check(getBundleFile(b))
		})
	}

	bundler, ok := bundlers[bundleType]
	if !ok {
		return fmt.Errorf("unknown bundle type: %s (valid: brew, js, python/py, rust, all)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	return bundler.Check(getBundleFile(bundler))
}

func runCleanup(bundlers map[string]Bundler, bundleType string) error {
	if bundleType == "all" {
		return runAll(bundlers, func(b Bundler) error {
			return b.Cleanup(getBundleFile(b), verboseFlag, forceFlag)
		})
	}

	bundler, ok := bundlers[bundleType]
	if !ok {
		return fmt.Errorf("unknown bundle type: %s (valid: brew, js, python/py, rust, all)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	return bundler.Cleanup(getBundleFile(bundler), verboseFlag, forceFlag)
}

func runAll(bundlers map[string]Bundler, fn func(Bundler) error) error {
	order := []string{"brew", "js", "python", "rust"}
	var errs []error

	for _, name := range order {
		bundler := bundlers[name]
		fmt.Printf("\n=== %s ===\n", name)

		if err := bundler.CheckDeps(); err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", name, err)
			continue
		}

		if err := fn(bundler); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", name, err)
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d bundler(s) failed", len(errs))
	}
	return nil
}

func getBundleFile(bundler Bundler) string {
	if fileFlag != "" {
		return fileFlag
	}
	return bundler.DefaultFile()
}

func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return "vim"
	}
	return editor
}

func openInEditor(file string) error {
	editor := getEditor()
	cmd := exec.Command(editor, file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runIterative(bundlers map[string]Bundler, bundleType string) error {
	if bundleType == "all" {
		return fmt.Errorf("iterative mode not supported for 'all'")
	}

	bundler, ok := bundlers[bundleType]
	if !ok {
		return fmt.Errorf("unknown bundle type: %s (valid: brew, js, python/py, rust)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	bundleFile := expandPath(getBundleFile(bundler))

	// Always open the main bundle file
	if err := openInEditor(bundleFile); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	fmt.Println("\n=== Syncing ===")
	return bundler.Sync(bundleFile, verboseFlag, forceFlag, noUpdateFlag, tapsFlag, masFlag)
}
