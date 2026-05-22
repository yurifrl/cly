package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	agentsession "github.com/yurifrl/cly/modules/agent-session"
	"github.com/yurifrl/cly/modules/agents"
	"github.com/yurifrl/cly/modules/aliases"
	"github.com/yurifrl/cly/modules/backup"
	"github.com/yurifrl/cly/modules/beads"
	"github.com/yurifrl/cly/modules/bundle"
	"github.com/yurifrl/cly/modules/claude"
	claudetasks "github.com/yurifrl/cly/modules/claude-tasks"
	"github.com/yurifrl/cly/modules/completion"
	"github.com/yurifrl/cly/modules/config"
	"github.com/yurifrl/cly/modules/demo"
	"github.com/yurifrl/cly/modules/diff2"
	"github.com/yurifrl/cly/modules/dotfiles"
	gitcommits "github.com/yurifrl/cly/modules/git-commits"
	"github.com/yurifrl/cly/modules/helpy"
	llmchat "github.com/yurifrl/cly/modules/llm-chat"
	"github.com/yurifrl/cly/modules/mcp"
	"github.com/yurifrl/cly/modules/memwatch"
	"github.com/yurifrl/cly/modules/notify"
	"github.com/yurifrl/cly/modules/piwrap"
	"github.com/yurifrl/cly/modules/obsidian-tools"
	"github.com/yurifrl/cly/modules/scraper/cmd"
	"github.com/yurifrl/cly/modules/statusline"
	"github.com/yurifrl/cly/modules/update"
	"github.com/yurifrl/cly/modules/uuid"
	"github.com/yurifrl/cly/modules/zl"
	"github.com/yurifrl/cly/modules/oi"
	"github.com/yurifrl/cly/modules/py"
	"github.com/yurifrl/cly/modules/skills"
	"github.com/yurifrl/cly/modules/zs"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var versionFlag bool

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
		if versionFlag {
			printVersion()
			os.Exit(0)
		}
		_, err := pkgconfig.Load()
		return err
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "version for cly")
	RootCmd.SetVersionTemplate(versionString())
	completion.Register(RootCmd)
	agents.Register(RootCmd)
	aliases.Register(RootCmd)
	backup.Register(RootCmd)
	beads.Register(RootCmd)
	claude.Register(RootCmd)
	agentsession.Register(RootCmd)
	claudetasks.Register(RootCmd)
	uuid.Register(RootCmd)
	demo.Register(RootCmd)
	diff2.Register(RootCmd)
	config.Register(RootCmd)
	dotfiles.Register(RootCmd)
	gitcommits.Register(RootCmd)
	helpy.Register(RootCmd)
	llmchat.Register(RootCmd)
	mcp.Register(RootCmd)
	bundle.Register(RootCmd)
	notify.Register(RootCmd)
	piwrap.Register(RootCmd)
	memwatch.Register(RootCmd)
	obsidiantools.Register(RootCmd)
	statusline.Register(RootCmd)
	update.Register(RootCmd)
	zl.Register(RootCmd)
	oi.Register(RootCmd)
	py.Register(RootCmd)
	skills.Register(RootCmd)
	zs.Register(RootCmd)
	cmd.Register(RootCmd)
}

func Execute() error {
	return RootCmd.Execute()
}

func versionString() string {
	buildName := GenerateBuildName(BuildTime)
	return fmt.Sprintf("cly %s (%s)\nBuilt: %s\n", Version, buildName, BuildTime)
}

func printVersion() {
	fmt.Print(versionString())
}
