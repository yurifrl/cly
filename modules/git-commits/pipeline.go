package gitcommits

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/ai"
	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/llm"
	"github.com/yurifrl/cly/pkg/style"
)

type pipelineOpts struct {
	DryRun   bool
	Yes      bool
	All      bool
	JSON     bool
	NoVerify bool
	Push     bool
	Strategy string
	Prompt   string
	Ignored  bool
	NoSubmodule bool
}

const (
	StrategyFile = "file"
	StrategyLine = "line"
)

func runPipeline(cmd *cobra.Command, opts pipelineOpts) error {
	// Validate strategy
	strategy := opts.Strategy
	if strategy == "" {
		strategy = StrategyFile
	}
	if strategy != StrategyFile && strategy != StrategyLine {
		return fmt.Errorf("unknown strategy %q — use 'file' or 'line'", strategy)
	}

	// Verify git binary exists
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH — install git first")
	}

	// Submodules: if any have uncommitted changes, optionally commit them first.
	if !opts.NoSubmodule {
		if statuses := submoduleStatuses(); len(statuses) > 0 {
			var commitable []string
			for _, s := range statuses {
				if !s.HasStaged && !opts.All {
					fmt.Println(style.SubtleStyle.Render(fmt.Sprintf("⏭  submodule %s: nothing staged, skipping (%d unstaged/untracked)", s.Path, len(s.Unstaged))))
					continue
				}
				commitable = append(commitable, s.Path)
			}
			if len(commitable) > 0 {
				commit := opts.Yes || confirmSubmoduleCommit(commitable)
				if commit {
					if err := commitSubmodules(commitable, opts); err != nil {
						return fmt.Errorf("committing submodules: %w", err)
					}
				} else {
					fmt.Println(style.SubtleStyle.Render("   skipping submodule commits"))
				}
			}
		}

		// Stage any submodule whose pointer is unstaged in the parent so it
		// gets included in the parent commit plan. Covers the case where the
		// submodule was already committed (by us or another session) but its
		// new pointer was never `git add`-ed in the parent.
		if staged, err := stageUnstagedSubmodulePointers(); err == nil && len(staged) > 0 {
			for _, p := range staged {
				fmt.Println(style.SubtleStyle.Render(fmt.Sprintf("   staged submodule pointer: %s", p)))
			}
		}
	}

	// Load config
	batchSize := defaultBatchSize
	timeout := defaultTimeout
	maxGroups := 8
	customPrompt := opts.Prompt
	var ignorePatterns []string

	cfg := config.Get()
	if cfg != nil {
		if mod, ok := cfg.Modules["git-commits"]; ok {
			if bs, ok := mod["batch_size"]; ok {
				if v, ok := bs.(int); ok && v > 0 {
					batchSize = v
				}
			}
			if t, ok := mod["timeout"]; ok {
				if v, ok := t.(int); ok && v > 0 {
					timeout = time.Duration(v) * time.Millisecond
				}
			}
			if mg, ok := mod["max_groups"]; ok {
				if v, ok := mg.(int); ok && v > 0 {
					maxGroups = v
				}
			}
			if sp, ok := mod["split_prompt"]; ok {
				if v, ok := sp.(string); ok && v != "" && customPrompt == "" {
					customPrompt = v
				}
			}
			if ig, ok := mod["ignore"]; ok {
				switch v := ig.(type) {
				case []interface{}:
					for _, item := range v {
						if s, ok := item.(string); ok {
							ignorePatterns = append(ignorePatterns, s)
						}
					}
				case []string:
					ignorePatterns = append(ignorePatterns, v...)
				case string:
					for _, s := range strings.Split(v, ",") {
						if s = strings.TrimSpace(s); s != "" {
							ignorePatterns = append(ignorePatterns, s)
						}
					}
				}
			}
		}
	}

	// Create LLM client
	llmCfg := resolveLLMConfig(cfg)
	client, err := llm.NewClient(llmCfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w\nSet ANTHROPIC_API_KEY or OPENAI_API_KEY", err)
	}

	// Step 1: Analyze changeset
	fmt.Printf("⏳ Analyzing changes... (AI: %s)\n", activeLLMSummary())
	preStaged, err := stagedFiles()
	if err != nil {
		return fmt.Errorf("failed to inspect staged files: %w", err)
	}
	cs, err := GetChangeset(opts.All)
	if err != nil {
		return err
	}
	fmt.Printf("   Found %d file(s)\n", len(cs.Files))

	if !opts.Ignored && len(ignorePatterns) > 0 {
		removed, err := filterIgnored(cs, ignorePatterns, preStaged)
		if err != nil {
			return err
		}
		if len(removed) > 0 {
			fmt.Printf("   %s %d ignored file(s) (use --ignored to include):\n", style.YellowStyle.Render("⏭"), len(removed))
			for _, p := range removed {
				fmt.Printf("     - %s\n", p)
			}
			if len(cs.Files) == 0 {
				return fmt.Errorf("no staged changes after applying ignore patterns")
			}
		}
	}

	// Step 2: Batch
	var batches []Batch
	if strategy == StrategyLine {
		batches = BuildLineBatches(cs, batchSize)
	} else {
		batches = BuildBatches(cs, batchSize)
	}
	if len(batches) > 1 {
		fmt.Printf("   Split into %d batches\n", len(batches))
	}

	// Step 3: Plan via LLM (with revision loop)
	ctx := context.Background()
	planCfg := PlannerConfig{
		Timeout:      timeout,
		CustomPrompt: customPrompt,
		Strategy:     strategy,
		MaxGroups:    maxGroups,
	}

	var plan *CommitPlan
	for {
		fmt.Println(style.BlueStyle.Render("🤖 Planning split..."))

		raw, err := PlanSplit(ctx, batches, client, planCfg)
		if err != nil {
			// Fallback to single commit
			fmt.Println(style.YellowStyle.Render("⚠️  Split planning failed, falling back to single commit"))
			raw, err = GenerateFallbackMessage(ctx, cs, client, timeout)
			if err != nil {
				return fmt.Errorf("fallback also failed: %w", err)
			}
		}

		// Step 4: Validate and heal
		var validateErr error
		skipHeal := customPrompt != ""
		if strategy == StrategyLine {
			plan, validateErr = ValidateLinePlan(raw, cs)
		} else {
			plan, validateErr = ValidatePlan(raw, cs, skipHeal)
		}
		if validateErr != nil {
			fmt.Println(style.YellowStyle.Render("⚠️  Plan validation failed, falling back to single commit"))
			raw, err = GenerateFallbackMessage(ctx, cs, client, timeout)
			if err != nil {
				return fmt.Errorf("fallback also failed: %w", err)
			}
			plan, err = ValidatePlan(raw, cs, skipHeal)
			if err != nil {
				return fmt.Errorf("even fallback plan failed validation: %w", err)
			}
		}

		// Step 5: Preview
		if opts.JSON {
			return RenderJSON(plan)
		}

		fmt.Print(RenderPlan(plan))

		if opts.DryRun {
			fmt.Println(style.YellowStyle.Render("(dry run — no changes made)"))
			return nil
		}

		if opts.Yes {
			break // Auto-confirm, proceed to execute
		}

		// Interactive: Y/n/revise
		result := Confirm()
		switch result.Action {
		case ConfirmYes:
			goto execute
		case ConfirmNo:
			fmt.Println("Aborted.")
			return nil
		case ConfirmRevise:
			// Append the user's guidance and re-plan
			fmt.Println()
			planCfg.CustomPrompt = buildRevisionPrompt(planCfg.CustomPrompt, plan, result.Prompt)
			continue
		}
	}

execute:
	// Step 6: Execute
	fmt.Println(style.BlueStyle.Render(fmt.Sprintf("⚡ Creating %d commits...", len(plan.Groups))))
	var results []CommitResult
	if strategy == StrategyLine {
		results, err = ExecuteLine(plan, cs, opts.NoVerify)
	} else {
		results, err = Execute(plan, opts.NoVerify)
	}
	// Don't return err yet — print results first, then report at the end

	// Print results
	fmt.Println()
	successCount := 0
	for _, r := range results {
		if r.Skipped {
			fmt.Printf("%s %s (%d files): %v\n",
				style.RedStyle.Render("✗ SKIP"),
				r.Title,
				r.Files,
				r.Err)
		} else {
			fmt.Printf("%s %s (%d files)\n",
				style.GreenStyle.Render("✓ "+r.SHA),
				r.Title,
				r.Files)
			successCount++
		}
	}
	fmt.Printf("\n%s\n", style.GreenStyle.Render(fmt.Sprintf("Done! Created %d/%d commits.", successCount, len(results))))

	// Push if requested
	if opts.Push {
		branch, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		fmt.Println(style.BlueStyle.Render(fmt.Sprintf("🚀 Pushing to %s...", branch)))
		if out, err := gitExec("push", "origin", branch); err != nil {
			return fmt.Errorf("git push failed: %s: %w", strings.TrimSpace(out), err)
		}
		fmt.Println(style.GreenStyle.Render("✓ Pushed!"))
	}

	return err
}

// resolveLLMConfig delegates to `pkg/ai`. The `cfg` arg is kept for
// back-compat but ignored; pkg/ai always reads the live process config.
// commit-specific AI overrides live under `modules.git-commits.ai`.
func resolveLLMConfig(_ *config.Config) llm.Config {
	r := ai.LoadConfigWith(ai.LookupModuleOverride("git-commits"))
	if r == nil {
		return llm.Config{}
	}
	return llm.Config{
		Provider:  llm.Provider(r.Provider),
		Model:     r.Model,
		APIKey:    r.APIKey,
		APIKeyEnv: r.APIKeyEnv,
		BaseURL:   r.BaseURL,
	}
}

// buildRevisionPrompt creates a new custom prompt that includes the previous plan
// as context so the LLM can revise it based on user feedback.
func buildRevisionPrompt(existingPrompt string, previousPlan *CommitPlan, userFeedback string) string {
	var b strings.Builder

	if existingPrompt != "" {
		b.WriteString(existingPrompt)
		b.WriteString("\n\n")
	}

	// Show the previous plan so the LLM knows what to revise
	b.WriteString("PREVIOUS PLAN (revise based on user feedback below):\n")
	for i, g := range previousPlan.Groups {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, g.Title))
		for _, f := range g.Files {
			b.WriteString(fmt.Sprintf("     - %s\n", f.Path))
		}
	}

	b.WriteString("\nUSER FEEDBACK: ")
	b.WriteString(userFeedback)

	return b.String()
}
