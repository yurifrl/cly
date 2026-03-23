package gitcommits

import (
	"github.com/spf13/cobra"
)

var (
	dryRunFlag   bool
	yesFlag      bool
	allFlag      bool
	jsonFlag     bool
	noVerifyFlag bool
	yoloFlag     bool
	strategyFlag string
	promptFlag   string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "git-commits",
		Aliases: []string{"gc"},
		Short:   "AI-powered split commits",
		Long: `Analyze staged changes, split into logical atomic commits using AI,
and execute them with conventional commit messages.

The command analyzes your git diff, groups related changes, generates
commit messages, and creates multiple focused commits automatically.`,
		RunE: run,
	}

	cmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "d", false, "Show plan without executing")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Execute without confirmation")
	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Stage all changes (including untracked) before splitting")
	cmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output raw JSON plan")
	cmd.Flags().BoolVarP(&noVerifyFlag, "no-verify", "n", false, "Bypass pre-commit hooks")
	cmd.Flags().BoolVar(&yoloFlag, "yolo", false, "Stage all, auto-confirm, push to current branch")
	cmd.Flags().StringVarP(&strategyFlag, "strategy", "s", "file", "Split strategy: file (whole files) or line (hunk-level)")
	cmd.Flags().StringVarP(&promptFlag, "prompt", "p", "", "Custom prompt to append to system prompt")

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	return runPipeline(cmd, pipelineOpts{
		DryRun:   dryRunFlag,
		Yes:      yesFlag || yoloFlag,
		All:      allFlag || yoloFlag,
		JSON:     jsonFlag,
		NoVerify: noVerifyFlag,
		Prompt:   promptFlag,
		Push:     yoloFlag,
		Strategy: strategyFlag,
	})
}
