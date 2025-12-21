package bundle

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/store"
)

var (
	fileFlag    string
	verboseFlag bool
	noItFlag    bool
	forceFlag   bool
	tapsFlag    bool
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
			"go":     NewGoBundler(s),
			"js":     NewJsBundler(s),
			"python": NewPythonBundler(s),
		}
		cleanup := func() {
			if err := s.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close store: %v\n", err)
			}
		}
		return bundlers, cleanup, nil
	}

	cmd := &cobra.Command{
		Use:   "bundle [type]",
		Short: "Edit and sync packages from declarative bundle files",
		Long: `Unified declarative package management for brew, go, js, and python.

Opens bundle file in $EDITOR, syncs after save, prompts to continue editing.

Types:
  brew    Sync Homebrew packages from Brewfile (default)
  go      Sync Go binaries from Gofile
  js      Sync JavaScript packages from Jsfile
  python  Sync Python tools from Pythonfile`,
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
			if noItFlag {
				return runSync(bundlers, bundleType)
			}
			return runIterative(bundlers, bundleType)
		},
	}

	cmd.Flags().StringVarP(&fileFlag, "file", "f", "", "override bundle file path")
	cmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "show detailed output")
	cmd.Flags().BoolVar(&noItFlag, "no-it", false, "skip interactive editor mode, just sync")
	cmd.Flags().BoolVar(&forceFlag, "force", true, "force reinstall packages even if already installed")
	cmd.Flags().BoolVar(&tapsFlag, "taps", false, "run Brewfile.taps first (brew only)")

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
	if bundleType == "all" {
		return runAll(bundlers, func(b Bundler) error {
			return b.Sync(getBundleFile(b), verboseFlag, forceFlag, tapsFlag)
		})
	}

	bundler, ok := bundlers[bundleType]
	if !ok {
		return fmt.Errorf("unknown bundle type: %s (valid: brew, go, js, python, all)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	return bundler.Sync(getBundleFile(bundler), verboseFlag, forceFlag, tapsFlag)
}

func runCheck(bundlers map[string]Bundler, bundleType string) error {
	if bundleType == "all" {
		return runAll(bundlers, func(b Bundler) error {
			return b.Check(getBundleFile(b))
		})
	}

	bundler, ok := bundlers[bundleType]
	if !ok {
		return fmt.Errorf("unknown bundle type: %s (valid: brew, go, js, python, all)", bundleType)
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
		return fmt.Errorf("unknown bundle type: %s (valid: brew, go, js, python, all)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	return bundler.Cleanup(getBundleFile(bundler), verboseFlag, forceFlag)
}

func runAll(bundlers map[string]Bundler, fn func(Bundler) error) error {
	order := []string{"brew", "go", "js", "python"}
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
		return fmt.Errorf("unknown bundle type: %s (valid: brew, go, js, python)", bundleType)
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	bundleFile := expandPath(getBundleFile(bundler))

	if err := openInEditor(bundleFile); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	fmt.Println("\n=== Syncing ===")
	return bundler.Sync(bundleFile, verboseFlag, forceFlag, tapsFlag)
}
