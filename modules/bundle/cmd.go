package bundle

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	editFlag   bool
	noEditFlag bool
	dryRunFlag bool
	fileFlag   string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "bundle [type]",
		Short: "Sync packages from declarative files",
		Long: `Unified declarative package management for multiple ecosystems.

Available types:
  brew    (default) Homebrew packages from ~/.config/Brewfile
  go      Go binaries from ~/.config/Gofile
  js      JavaScript packages from ~/.config/Jsfile
  python  Python tools from ~/.config/Pythonfile`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runBundle,
		Example: `  cly bundle           # sync brew packages (default)
  cly bundle go        # sync go binaries
  cly bundle --dry-run # preview changes without applying`,
	}

	cmd.Flags().BoolVarP(&editFlag, "edit", "e", true, "open bundle file in editor first")
	cmd.Flags().BoolVar(&noEditFlag, "no-edit", false, "skip editor, just sync")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "show changes without applying")
	cmd.Flags().StringVarP(&fileFlag, "file", "f", "", "override bundle file path")

	parent.AddCommand(cmd)
}

func runBundle(cmd *cobra.Command, args []string) error {
	bundleType := "brew"
	if len(args) > 0 {
		bundleType = args[0]
	}

	bundler, err := getBundler(bundleType)
	if err != nil {
		return err
	}

	if err := bundler.CheckDeps(); err != nil {
		return err
	}

	bundleFile := fileFlag
	if bundleFile == "" {
		bundleFile = bundler.DefaultFile()
	}

	if _, err := os.Stat(bundleFile); os.IsNotExist(err) {
		return fmt.Errorf("bundle file not found: %s", bundleFile)
	}

	// Open editor unless --no-edit
	if editFlag && !noEditFlag {
		if err := openEditor(bundleFile); err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
	}

	return bundler.Sync(bundleFile, dryRunFlag)
}

func getBundler(bundleType string) (Bundler, error) {
	switch bundleType {
	case "brew":
		return &BrewBundler{}, nil
	case "go":
		return &GoBundler{}, nil
	case "js":
		return &JsBundler{}, nil
	case "python":
		return &PythonBundler{}, nil
	default:
		return nil, fmt.Errorf("unknown bundle type: %s (valid: brew, go, js, python)", bundleType)
	}
}

func openEditor(file string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
