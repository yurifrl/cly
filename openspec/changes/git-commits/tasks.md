## 1. Foundation — pkg/llm extension

- [x] 1.1 Add `Complete(ctx, systemPrompt, messages) (string, error)` method to `pkg/llm.Client` interface
- [x] 1.2 Implement `Complete` for `anthropicClient` using non-streaming `Messages.New`
- [x] 1.3 Implement `Complete` for `openaiClient` using non-streaming `Chat.Completions.New`
- [x] 1.4 Add tests for `Complete` on both providers (mock or integration)

## 2. Module scaffold

- [x] 2.1 Create `modules/git-commits/cmd.go` with `Register(parent)`, command `git-commits`, alias `gc`
- [x] 2.2 Add flags: `--dry-run/-d`, `--yes/-y`, `--all/-a`, `--json/-j`, `--no-verify/-n`, `--prompt/-p`
- [x] 2.3 Register module in `cmd/root.go`
- [x] 2.4 Verify `cly git-commits --help` and `cly gc --help` work

## 3. Changeset analysis

- [x] 3.1 Create `modules/git-commits/changeset.go` — types: `FileChange{Path, OldPath, Status, Hunks}`, `Changeset`
- [x] 3.2 Implement `ParseDiff(diffOutput string) Changeset` — parse `git diff --cached` (or `HEAD`) output into structured changeset
- [x] 3.3 Implement `GetChangeset(all bool) (Changeset, error)` — runs `git diff` commands, handles `--all` flag staging, detects renames
- [x] 3.4 Handle untracked files when `--all` is set (synthetic diff or pre-stage with `git add .`)
- [x] 3.5 Add tests: parse staged diffs, renames, additions, deletions, empty changeset error

## 4. Batching

- [x] 4.1 Create `modules/git-commits/batcher.go` — `BuildBatches(changeset, batchSize int) []Batch`
- [x] 4.2 Implement greedy file packing by character budget (default 40,000 chars)
- [x] 4.3 Generate per-file analysis text (path, status, hunk summary, optional full diff if under secondary limit)
- [x] 4.4 Add tests: single batch for small changeset, multiple batches for large, respects custom batch size

## 5. LLM planning

- [x] 5.1 Create `modules/git-commits/planner.go` — system prompt with grouping heuristics and JSON schema
- [x] 5.2 Implement `PlanSplit(ctx, batches, client) (RawPlan, error)` — sends all batches in parallel via `errgroup`
- [x] 5.3 Scale `maxOutputTokens` based on analysis size (4K / 8K / 16K tiers)
- [x] 5.4 Implement JSON extraction from LLM response (strip markdown fences, find outermost `{}`)
- [x] 5.5 Implement merge of multi-batch results into single `{ "groups": [...] }`
- [x] 5.6 Implement fallback: on total failure, generate single conventional commit message
- [x] 5.7 Add 30s per-request timeout
- [x] 5.8 Add tests: JSON parsing, batch merging, fallback on failure

## 6. Validation and auto-healing

- [x] 6.1 Create `modules/git-commits/validator.go` — `ValidatePlan(rawPlan, changeset) (CommitPlan, error)`
- [x] 6.2 Resolve file references (match by path or oldPath for renames)
- [x] 6.3 Auto-heal: deduplicate files across groups (keep first occurrence)
- [x] 6.4 Auto-heal: assign uncovered files to group with longest shared directory prefix
- [x] 6.5 Reject empty plans (zero groups after healing)
- [x] 6.6 Add tests: dedup, uncovered assignment, empty plan rejection, rename resolution

## 7. Preview

- [x] 7.1 Create `modules/git-commits/preview.go` — `RenderPlan(plan CommitPlan) string` styled with `pkg/style`
- [x] 7.2 Implement numbered commit display: title, summary, file list with status indicators
- [x] 7.3 Implement `--json` output mode (raw JSON to stdout)
- [x] 7.4 Implement `--dry-run` mode (print plan, exit 0, no git mutation)
- [x] 7.5 Implement interactive confirmation prompt (`Execute? [Y/n]`) reading from stdin
- [x] 7.6 Implement `--yes` mode (skip prompt, execute immediately)

## 8. Execution

- [x] 8.1 Create `modules/git-commits/executor.go` — `Execute(plan CommitPlan, noVerify bool) ([]CommitResult, error)`
- [x] 8.2 Record `originalHead` via `git rev-parse HEAD` and save full diff for recovery
- [x] 8.3 Implement `git reset` to unstage everything before execution loop
- [x] 8.4 Implement per-group staging: `git add` for A/M, `git rm --cached` for D, both for R
- [x] 8.5 Implement `git commit -m <title> [-m <body>] [--no-verify]` per group
- [x] 8.6 Implement rollback: `git reset --soft <originalHead>` → `git reset` → `git apply --cached < savedDiff>`
- [x] 8.7 Ensure saved diff ends with `\n` (prevent corrupt patch)
- [x] 8.8 On rollback failure, print original HEAD SHA for manual recovery
- [x] 8.9 Add tests: multi-commit execution, rollback on failure, no-verify passthrough

## 9. Config integration

- [x] 9.1 Add `modules.git-commits` section to config defaults (provider, model, batch-size, timeout, split-prompt)
- [x] 9.2 Wire config values into planner and batcher (batch size, timeout, custom prompt)
- [x] 9.3 Support `--prompt` flag to override/append to system prompt

## 10. Wire it all together

- [x] 10.1 Implement main `RunE` in `cmd.go`: changeset → batch → plan → validate → preview → execute pipeline
- [x] 10.2 Add progress output: "Analyzing N files...", "Planning split...", "Creating N commits..."
- [x] 10.3 Verify `git` binary exists at startup, fail fast with clear error if missing
- [x] 10.4 End-to-end test: create temp git repo, stage mixed changes, run command, verify commits created
