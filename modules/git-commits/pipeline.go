package gitcommits

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
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

	// Load config
	batchSize := defaultBatchSize
	timeout := defaultTimeout
	customPrompt := opts.Prompt

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
			if sp, ok := mod["split_prompt"]; ok {
				if v, ok := sp.(string); ok && v != "" && customPrompt == "" {
					customPrompt = v
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
	cs, err := GetChangeset(opts.All)
	if err != nil {
		return err
	}
	fmt.Printf("   Found %d file(s)\n", len(cs.Files))

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
		if strategy == StrategyLine {
			plan, validateErr = ValidateLinePlan(raw, cs)
		} else {
			plan, validateErr = ValidatePlan(raw, cs)
		}
		if validateErr != nil {
			fmt.Println(style.YellowStyle.Render("⚠️  Plan validation failed, falling back to single commit"))
			raw, err = GenerateFallbackMessage(ctx, cs, client, timeout)
			if err != nil {
				return fmt.Errorf("fallback also failed: %w", err)
			}
			plan, err = ValidatePlan(raw, cs)
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

// resolveLLMConfig builds LLM config from cly config or env defaults.
func resolveLLMConfig(cfg *config.Config) llm.Config {
	// Default: OpenAI (override via modules.git-commits.ai.provider in config)
	llmCfg := llm.Config{
		Provider: llm.ProviderOpenAI,
	}

	if cfg != nil {
		if mod, ok := cfg.Modules["git-commits"]; ok {
			if ai, ok := mod["ai"]; ok {
				if aiMap, ok := ai.(map[string]interface{}); ok {
					if p, ok := aiMap["provider"].(string); ok {
						llmCfg.Provider = llm.Provider(p)
					}
					if m, ok := aiMap["model"].(string); ok {
						llmCfg.Model = m
					}
					if k, ok := aiMap["api_key"].(string); ok {
						llmCfg.APIKey = k
					}
					if ke, ok := aiMap["api_key_env"].(string); ok {
						llmCfg.APIKeyEnv = ke
					}
				}
			}
		}
	}

	return llmCfg
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
