package skills

import (
	"embed"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/embedfs"
)

//go:embed all:embedded
var embedded embed.FS

const defaultTarget = "~/.agents/skills/"

func installCmd() *cobra.Command {
	var target string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "install [name...]",
		Short: "Install embedded skills to --target (default ~/.agents/skills/). Optional names to cherry-pick; default is all.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, err := embedfs.ResolveTarget(target)
			if err != nil {
				return err
			}
			return embedfs.InstallSelected(embedded, "embedded", dest, args, dryRun, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&target, "target", defaultTarget, "Install destination directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be written without touching the filesystem")
	return cmd
}
