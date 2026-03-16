package zs

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/completion"
)

func Register(parent *cobra.Command) {
	completion.RegisterAlias("zs", GenerateCompletionsString())

	var explicitLayout string
	var noLayout bool
	var dryRun bool
	var debug bool
	var sessionMode bool
	var tabMode bool

	cmd := &cobra.Command{
		Use:          "zs [session-name]",
		Short:        "Smart Zellij sessionizer",
		Long:         "Interactive Zellij session picker inspired by zellij-smart-sessionizer.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			providedName := ""
			if len(args) > 0 {
				providedName = args[0]
			}

			debug = debug || envBool("ZS_DEBUG")
			dryRun = dryRun || envBool("ZS_DRY_RUN")
			sessionMode = sessionMode || envBool("ZS_SESSION_MODE")
			tabMode = tabMode || envBool("ZS_TAB_MODE")

			if sessionMode && tabMode {
				return fmt.Errorf("--session-mode and --tab-mode are mutually exclusive")
			}

			configureRuntime(dryRun, debug, sessionMode, tabMode)

			if err := run(providedName, explicitLayout, noLayout); err != nil {
				if errors.Is(err, ErrSelectionCanceled) {
					debugf("selection canceled")
					return nil
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&explicitLayout, "layout", "l", "", "Layout name or path")
	cmd.Flags().BoolVar(&noLayout, "no-layout", false, "Use the default layout without prompting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what zs would do without executing zellij")
	cmd.Flags().BoolVar(&debug, "debug", false, "Print debug information (or set ZS_DEBUG=1)")
	cmd.Flags().BoolVar(&sessionMode, "session-mode", false, "Force session attach/create behavior even inside Zellij")
	cmd.Flags().BoolVar(&tabMode, "tab-mode", false, "Force tab behavior even outside Zellij")

	parent.AddCommand(cmd)
}

func run(providedName, explicitLayout string, noLayout bool) error {
	debugf("start provided=%q explicitLayout=%q noLayout=%t dryRun=%t", providedName, explicitLayout, noLayout, runtimeCfg.DryRun)

	if err := ensureDependencies(); err != nil {
		return err
	}

	inside := IsInsideZellij()
	mode := "session"
	switch {
	case runtimeCfg.ForceSessionMode:
		mode = "session"
	case runtimeCfg.ForceTabMode:
		mode = "tab"
	case inside:
		mode = "tab"
	default:
		mode = "session"
	}

	debugf("inside_zellij=%t mode=%s", inside, mode)
	if mode == "tab" {
		return runInside(providedName, explicitLayout, noLayout)
	}
	return runOutside(providedName, explicitLayout, noLayout)
}

func runOutside(providedName, explicitLayout string, noLayout bool) error {
	sessions, err := ListSessionNames()
	if err != nil {
		return err
	}
	debugf("outside sessions count=%d sample=%s", len(sessions), summarizeStrings(sessions, 10))

	dirs, err := ListZoxideDirs()
	if err != nil {
		return err
	}
	debugf("outside zoxide_dirs count=%d sample=%s", len(dirs), summarizeStrings(dirs, 10))

	selection, err := SelectSessionTarget(sessions, dirs)
	if err != nil {
		return err
	}
	debugf("outside selection kind=%s value=%q", selection.Kind, selection.Value)

	if selection.Kind == selectionKindSession {
		debugf("attaching selected session=%q", selection.Value)
		return AttachSession(selection.Value)
	}

	sessionName := SessionNameForDir(selection.Value, providedName)
	debugf("derived session_name=%q", sessionName)
	if SessionExists(sessionName, sessions) {
		debugf("session exists, attaching session=%q", sessionName)
		return AttachSession(sessionName)
	}

	layout, err := ResolveLayout(explicitLayout, noLayout, false, sessionName)
	if err != nil {
		return err
	}
	debugf("resolved layout=%q", layout)

	return CreateSession(sessionName, selection.Value, layout)
}

func runInside(providedName, explicitLayout string, noLayout bool) error {
	dirs, err := ListZoxideDirs()
	if err != nil {
		return err
	}
	debugf("inside zoxide_dirs count=%d sample=%s", len(dirs), summarizeStrings(dirs, 10))

	selection, err := SelectDirectory(dirs, "Select tab directory:")
	if err != nil {
		return err
	}
	debugf("inside selection kind=%s value=%q", selection.Kind, selection.Value)

	tabName := SessionNameForDir(selection.Value, providedName)
	debugf("derived tab_name=%q", tabName)
	layout, err := ResolveLayout(explicitLayout, noLayout, true, tabName)
	if err != nil {
		return err
	}
	debugf("resolved layout=%q", layout)

	if err := NewTab(tabName, selection.Value, layout); err != nil {
		return fmt.Errorf("create tab %q: %w", tabName, err)
	}
	if err := GoToTab(tabName); err != nil {
		return fmt.Errorf("focus tab %q: %w", tabName, err)
	}
	return nil
}

func GenerateCompletionsString() string {
	return `# Fish completions for zs (cly zs smart sessionizer)
complete -c zs -s l -l layout -r -d "Layout name or path"
complete -c zs -l no-layout -d "Use default layout"
complete -c zs -l dry-run -d "Print what zs would do without executing zellij"
complete -c zs -l debug -d "Print debug information"
complete -c zs -l session-mode -d "Force session attach/create behavior"
complete -c zs -l tab-mode -d "Force tab behavior"
`
}
