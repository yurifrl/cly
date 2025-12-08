package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/config"
	"github.com/yurifrl/cly/modules/demo"
	"github.com/yurifrl/cly/modules/uuid"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

var RootCmd = &cobra.Command{
	Use:   "cly",
	Short: style.TitleStyle.Render("Charm Libraries Showcase"),
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
	uuid.Register(RootCmd)
	demo.Register(RootCmd)
	config.Register(RootCmd)
}

func Execute() error {
	return RootCmd.Execute()
}
