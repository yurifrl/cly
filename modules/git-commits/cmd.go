package gitcommits

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/ai"
)

var (
	dryRunFlag      bool
	yesFlag         bool
	allFlag         bool
	jsonFlag        bool
	noVerifyFlag    bool
	yoloFlag        bool
	strategyFlag    string
	promptFlag      string
	ignoredFlag     bool
	noSubmoduleFlag bool
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:          "git-commits",
		Aliases:      []string{"gc"},
		Short:        "AI-powered split commits",
		RunE:         run,
		SilenceUsage: true,
	}

	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		c.Long = fmt.Sprintf(`Analyze staged changes, split into logical atomic commits using AI,
and execute them with conventional commit messages.

The command analyzes your git diff, groups related changes, generates
commit messages, and creates multiple focused commits automatically.

AI provider: %s (configure via modules.git-commits.ai in config)`,
			activeLLMSummary())
		c.Parent().HelpFunc()(c, args)
	})

	cmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "d", false, "Show plan without executing")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Execute without confirmation")
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Stage all changes (including untracked) before splitting")
	cmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output raw JSON plan")
	cmd.Flags().BoolVarP(&noVerifyFlag, "no-verify", "n", false, "Bypass pre-commit hooks")
	cmd.Flags().BoolVar(&yoloFlag, "yolo", false, "Stage all, auto-confirm, push to current branch")
	cmd.Flags().StringVarP(&strategyFlag, "strategy", "s", "file", "Split strategy: file (whole files) or line (hunk-level)")
	cmd.Flags().StringVarP(&promptFlag, "prompt", "p", "", "Custom prompt to append to system prompt")
	cmd.Flags().BoolVar(&ignoredFlag, "ignored", false, "Include files matching modules.git-commits.ignore patterns")
	cmd.Flags().BoolVar(&noSubmoduleFlag, "no-submodule", false, "Skip committing dirty submodules first")

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	return runPipeline(cmd, pipelineOpts{
		DryRun:      dryRunFlag,
		Yes:         yesFlag || yoloFlag,
		All:         allFlag || yoloFlag,
		JSON:        jsonFlag,
		NoVerify:    noVerifyFlag,
		Prompt:      promptFlag,
		Push:        yoloFlag,
		Strategy:    strategyFlag,
		Ignored:     ignoredFlag,
		NoSubmodule: noSubmoduleFlag,
	})
}

// activeLLMSummary returns a short human-readable description of the active LLM.
func activeLLMSummary() string {
	resolved := ai.LoadConfigWith(ai.LookupModuleOverride("git-commits"))
	if resolved == nil {
		return "disabled"
	}
	return fmt.Sprintf("%s / %s", resolved.Provider, resolved.Model)
}
