## Why

Running `git wip` (or similar) after a working session means manually reviewing diffs, deciding how to split changes, writing commit messages, and staging files one-by-one. This is tedious, error-prone, and breaks flow. An AI-powered command should analyze the changeset, propose an optimal multi-commit split plan, and execute it — all in under 5 seconds for typical changesets.

## What Changes

- New `cly git-commits` command (alias: `gc`) that analyzes staged/unstaged changes and produces atomic, well-messaged commits
- Uses `pkg/llm` (Anthropic/OpenAI) to analyze diffs, group related changes, and generate conventional commit messages
- Implements the split-commit pipeline from `docs/SPLIT-SPEC.md`: changeset → batch → plan → validate/heal → preview → execute
- Supports `--dry-run`, `--yes`, `--all`, `--json` flags for flexible workflows
- Designed for speed: parallel batch planning, minimal LLM round-trips, no TUI overhead in default mode
- Auto-heals LLM output (dedup files across groups, assign uncovered files)
- Safe rollback on execution failure

## Capabilities

### New Capabilities
- `git-commits`: AI-powered split-commit workflow — analyzes git diffs, batches large changesets, plans multi-commit splits via LLM, validates/heals the plan, previews, and executes atomic commits with rollback safety

### Modified Capabilities
<!-- None — this is a new standalone module -->

## Impact

- **New module**: `modules/git-commits/` (cmd.go, changeset.go, batcher.go, planner.go, executor.go, preview.go)
- **Dependencies**: Uses existing `pkg/llm` for AI calls, `pkg/config` for settings, `pkg/style` for output formatting
- **External**: Shells out to `git` for diff parsing, staging, and committing — no new Go dependencies needed
- **Config**: New `modules.git-commits` section in config.yaml (provider, model, batch-size, timeout, split-prompt)
- **Registration**: One new line in `cmd/root.go`
