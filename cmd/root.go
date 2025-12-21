package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/ai"
	"github.com/yurifrl/cly/modules/backup"
	"github.com/yurifrl/cly/modules/bundle"
	"github.com/yurifrl/cly/modules/claude"
	"github.com/yurifrl/cly/modules/config"
	"github.com/yurifrl/cly/modules/demo"
	"github.com/yurifrl/cly/modules/dotfiles"
	"github.com/yurifrl/cly/modules/helpy"
	"github.com/yurifrl/cly/modules/mcp"
	"github.com/yurifrl/cly/modules/notify"
	"github.com/yurifrl/cly/modules/scraper/cmd"
	"github.com/yurifrl/cly/modules/update"
	"github.com/yurifrl/cly/modules/uuid"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

var Version = "dev"

var RootCmd = &cobra.Command{
	Use:     "cly",
	Short:   style.TitleStyle.Render("Charm Libraries Showcase"),
	Version: Version,
	Long: `Interactive demos of Bubbletea, Bubbles, Huh, and Lipgloss.

Each command demonstrates a different Charm component:
  • spinner   - Animated loading spinners
  • textinput - Text input fields
  • list      - Selectable lists
  • table     - Data tables

Press 'q' or Ctrl+C to quit any demo.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := pkgconfig.Load()
		return err
	},
}

func init() {
	RootCmd.SetVersionTemplate(fmt.Sprintf("cly %s\n", Version))
	ai.Register(RootCmd)
	backup.Register(RootCmd)
	claude.Register(RootCmd)
	uuid.Register(RootCmd)
	demo.Register(RootCmd)
	config.Register(RootCmd)
	dotfiles.Register(RootCmd)
	helpy.Register(RootCmd)
	mcp.Register(RootCmd)
	bundle.Register(RootCmd)
	notify.Register(RootCmd)
	update.Register(RootCmd)
	cmd.Register(RootCmd)
}

func Execute() error {
	return RootCmd.Execute()
}
