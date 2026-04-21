package pi

import (
	"embed"
	"io"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/embedfs"
)

//go:embed all:embedded
var embedded embed.FS

const defaultTarget = "~/.pi/agent/extensions/"

// InstallExtensions installs embedded pi extensions to target (empty = default).
// Pass dryRun=true to preview. Output is written to out.
func InstallExtensions(target string, dryRun bool, out io.Writer) error {
	if target == "" {
		target = defaultTarget
	}
	dest, err := embedfs.ResolveTarget(target)
	if err != nil {
		return err
	}
	return embedfs.Install(embedded, "embedded", dest, dryRun, out)
}

func installCmd() *cobra.Command {
	var target string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install embedded pi extensions to --target (default ~/.pi/agent/extensions/)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return InstallExtensions(target, dryRun, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&target, "target", defaultTarget, "Install destination directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be written without touching the filesystem")
	return cmd
}
