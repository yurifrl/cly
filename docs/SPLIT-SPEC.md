# Split Commit — Architecture Specification

## Problem

A developer stages a large set of changes spanning multiple concerns — new features, renames, refactors, config updates, tests. They want atomic commits but manually splitting is tedious and error-prone. The tool should analyze the changeset, plan a split, and execute it.

## Overview

The feature has two phases: **planning** and **execution**. Planning uses an LLM to produce a structured split plan. Execution validates the plan, confirms with the user, and creates the commits. The two phases are decoupled — planning produces data, execution consumes it.

```
changeset → [batch] → [plan] → [validate + heal] → [preview] → [execute]
```

---

## 1. Changeset Analysis

### Input

The tool operates on a set of file changes. Each change has:

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | File path relative to repo root |
| `oldPath` | string? | Previous path (renames only) |
| `status` | A \| M \| D \| R | Added, Modified, Deleted, Renamed |
| `hunks` | Hunk[] | Individual change blocks within the file |

Each hunk has:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (e.g. `h1`, `h2`) |
| `oldStart` | int | Starting line in the old file |
| `newStart` | int | Starting line in the new file |
| `rangeLabel` | string | Human-readable label (e.g. `changed 10-15`) |

### Source modes

The changeset can come from:

- **Staged changes** — `git diff --cached` (default)
- **Unstaged changes** — `git diff` + untracked files
- **All tracked changes** — `git diff HEAD`

Untracked files need synthetic diff generation (construct a valid unified diff from file contents).

---

## 2. Batching

### Why

LLMs have context window limits. A 60-file changeset with diffs can exceed 100KB. A single request either truncates, times out, or produces garbage output.

### Strategy

Split the changeset into batches by **character budget**. Each batch contains a subset of files with their summaries and diffs, sized to fit within the model's effective input capacity.

```
total analysis text → split by file boundaries → N batches
```

### Batch construction

1. Split the full diff into per-file sections (split on `diff --git` boundaries)
2. For each file, estimate size = summary lines + diff content
3. Greedily pack files into batches until the character budget is reached
4. Each batch gets a prompt header: `"Batch N of M — plan groups for ONLY these files"`

### Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `batch-size` | 40,000 chars | Max characters per batch analysis text |

### Sizing rationale

- System prompt: ~3K chars
- Output budget: ~16K tokens ≈ 64K chars
- Input budget for 128K model: ~445K chars (conservative: 40K per batch)
- 40K works for 32K+ context models and produces 2-5 batches for large changesets

### Diff inclusion

Per batch, include the full diff only if it fits within a secondary limit (e.g. 32K chars). Otherwise, send only the file/hunk summary. The planner needs file paths and hunk IDs for grouping — the diff improves commit message quality but isn't required.

---

## 3. Planning

### Request

Each batch is sent to the LLM as a separate request. All batches run in **parallel** (`Promise.all` or equivalent).

### System prompt

The prompt must include:

1. **Task definition** — "Split changes into the smallest sensible set of logical commits"
2. **Grouping heuristics** (priority order):
   - Renames: old path deletion + new path addition = one group
   - Same directory prefix = likely same group
   - Implementation + test files = same group
   - Config + implementation = same group
   - Different types (feat vs fix vs chore) = separate groups
3. **Hard constraints**:
   - Every file must appear in exactly one group
   - Each file in only one group (no cross-group splitting)
   - Output must be valid JSON, no prose
4. **Commit message format** — conventional, gitmoji, or plain
5. **Output schema** — exact JSON shape with example

### Output schema

```json
{
  "groups": [
    {
      "title": "feat: add session management",
      "type": "feat",
      "summary": "Groups session-related changes.",
      "body": "optional multi-line body",
      "items": [
        { "file": "src/session.ts" },
        { "file": "src/config.ts", "hunks": ["h3", "h5"] }
      ]
    }
  ]
}
```

### Output token scaling

Scale `maxOutputTokens` with analysis size:

| Analysis size | Output tokens |
|--------------|---------------|
| < 8K chars | 4,000 |
| 8K–20K chars | 8,000 |
| > 20K chars | 16,000 |

### Timeout

Planning needs more time than single-message generation. Default: 30s (60s for local models). Configurable.

### Fallback

If planning fails (invalid JSON, timeout, all batches fail), fall back to a single-commit plan:
- Generate a standard commit message using the existing single-message flow
- Wrap all files in one group
- This guarantees the command always produces something usable

---

## 4. Plan Merging

When multiple batches return, merge their results:

1. Extract `groups` array from each batch response
2. Concatenate all groups into one flat array
3. Serialize as `{ "groups": [...all] }`
4. Pass to the parser/validator with the **full** changeset (not per-batch)

Invalid batch responses (malformed JSON) are silently skipped. If all batches fail, trigger the fallback.

---

## 5. Parsing & Normalization

Parse the merged JSON plan against the actual changeset:

1. **Extract JSON** — strip markdown fences, find outermost `{}`
2. **Validate structure** — groups array, each group has title/items
3. **Resolve files** — match each `file` reference to a real changeset entry (by path or oldPath for renames)
4. **Resolve hunks** — if specific hunk IDs are given, verify they exist. If omitted or `"all"`, include all hunks.
5. **Reject unknowns** — unknown file paths or hunk IDs = parsing error (triggers fallback)

### Output data model

```
CommitPlan {
  groups: CommitPlanGroup[]
}

CommitPlanGroup {
  title: string          — commit message first line
  type: string           — conventional type (feat, fix, etc.)
  summary: string        — human-readable explanation
  body?: string          — optional commit body
  files: CommitPlanFile[]
}

CommitPlanFile {
  path: string
  oldPath?: string
  status: A | M | D | R
  ranges: string[]       — human-readable hunk labels
  hunkIds: string[]      — actual hunk IDs for cross-reference
  wholeFile: boolean     — true if all hunks of the file are included
}
```

---

## 6. Validation & Auto-Healing

After parsing, validate the plan for executability. **Auto-heal** what can be fixed. **Fail** only on unrecoverable errors.

### Auto-heal: duplicate files across groups

LLMs frequently place renamed files in both a "rename" group and a "module" group. Fix: keep the file in whichever group it appeared first, remove from later groups. Drop groups that become empty.

### Auto-heal: uncovered files

LLMs occasionally miss files. Fix: for each uncovered file, find the group whose existing files share the longest directory prefix, and append the file there.

### Hard failure

The only unrecoverable error is an empty plan (zero groups after healing). Everything else is auto-fixable.

---

## 7. Preview

After validation, render the plan for the user:

```
Split plan — 4 commits:

1. refactor: rename claude-session to agent-session
   Groups the module rename and old path deletions.
  • R claude-session/cmd.go -> agent-session/cmd.go (changed 1-4)
  • D claude-session/sessions.go (removed 1-93)

2. feat: add agent session providers
   New provider abstraction for multi-agent support.
  • A agent-session/providers.go (added 1-164)
  • A agent-session/providers_test.go (added 1-34)

...
```

### Modes

| Mode | Behavior |
|------|----------|
| Default (interactive) | Show plan → confirm → execute |
| `--yes` | Show plan → execute immediately |
| `--dry-run` | Show plan → exit (no git mutation) |
| `--json` | Output raw JSON plan → exit (no git mutation) |
| Headless (CI) | Requires `--yes`; text output, no prompts |

---

## 8. Execution

### Preconditions

- At least one existing commit (cannot split an initial commit)
- Validated plan with at least one group

### Algorithm

```
1. Record originalHead = git rev-parse HEAD
2. Save the full diff text for recovery
3. git reset (unstage everything, working tree preserved)
4. For each group:
   a. Stage the group's files:
      - Added/Modified: git add <path>
      - Deleted: git rm --cached <path>
      - Renamed: git rm --cached <oldPath> + git add <newPath>
   b. git commit -m <title> [-m <body>] [--no-verify]
   c. Record the new short SHA
5. Return list of (title, sha, files) tuples
```

### Why this works

After `git reset`, the index matches HEAD but the working tree retains all modifications. Each `git add` re-stages specific files from the working tree. After committing group 1, HEAD advances — group 2's files are still modified relative to the new HEAD, so `git add` stages the correct diff.

### Recovery on failure

If any step fails mid-execution (e.g. `git add` on a missing file):

```
1. git reset --soft <originalHead>   — move HEAD back
2. git reset                         — unstage everything (index = originalHead)
3. git apply --cached < savedDiff    — re-stage original changes
```

**Critical detail**: the saved diff must end with a newline. Tools like `execa` strip trailing newlines from stdout, which produces a corrupt patch. Always ensure `diff.endsWith('\n')` before saving.

If recovery itself fails, report the original HEAD SHA so the user can manually `git reset --soft`.

---

## 9. Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `split-prompt` | string | — | Persistent custom prompt appended to the system prompt |
| `split-batch-size` | int | 40,000 | Max characters per planning batch |
| `timeout` | int (ms) | 30,000 | Per-request timeout |

CLI flags override config:

| Flag | Overrides |
|------|-----------|
| `--prompt` / `-p` | `split-prompt` config |
| `--type` / `-t` | `type` config |
| `--dry-run` / `-d` | — |
| `--yes` / `-y` | — |
| `--no-verify` / `-n` | — |
| `--all` / `-a` | — (stages all files including untracked) |
| `--json` / `-j` | — |
| `--body` / `-b` | — |

---

## 10. Staging Scope

The `--all` flag must use `git add .` (not `git add --update`). `--update` only stages changes to tracked files — it ignores new untracked files. `git add .` stages everything (new, modified, deleted) while respecting `.gitignore`.

---

## 11. Error Handling Summary

| Scenario | Behavior |
|----------|----------|
| No changes to split | Error: "No staged changes found" |
| No AI provider configured | Error: "Run setup" |
| AI returns invalid JSON | Retry via fallback (single commit message) |
| AI misses files | Auto-assign to closest directory group |
| AI duplicates files across groups | Auto-dedup (keep first occurrence) |
| All planning batches fail | Fallback to single commit |
| Some batches fail | Merge surviving batches, auto-assign gaps |
| Staging fails mid-execution | Rollback to original HEAD + re-stage |
| Rollback fails | Report original HEAD for manual recovery |
| No initial commit exists | Error: "Create an initial commit first" |

---

## 12. Definition of Done

### Functional

- [ ] `split` analyzes staged changes and creates multiple atomic commits
- [ ] `split --dry-run` shows the plan without any git mutation
- [ ] `split --all` stages all files (including untracked) before splitting
- [ ] `split --json` outputs the structured plan as JSON
- [ ] `split --yes` executes without interactive confirmation
- [ ] `split --no-verify` bypasses pre-commit hooks
- [ ] `split --prompt` accepts a custom planning prompt
- [ ] Large changesets (50+ files) are batched and planned in parallel
- [ ] Failed batches don't block the overall plan
- [ ] Duplicate files across groups are auto-resolved
- [ ] Uncovered files are auto-assigned to the nearest group
- [ ] Failed execution rolls back to the original state
- [ ] Custom prompt is configurable persistently via config file
- [ ] Batch size is configurable via config file

### Safety

- [ ] `--dry-run` makes zero git state changes (HEAD, index, working tree all unchanged)
- [ ] Execution failure restores original HEAD and staging
- [ ] No commit is created from unvalidated AI output
- [ ] Empty plans are rejected
- [ ] The tool never silently drops files — every staged file ends up in a commit

### Tests

- [ ] Multi-file split creates separate commits in correct order
- [ ] Added, modified, deleted, and renamed files handled correctly
- [ ] Commit body included when requested
- [ ] Failed execution recovers to original HEAD + original staging
- [ ] Repos without initial commits are rejected
- [ ] Duplicate files across groups are deduplicated
- [ ] Uncovered files are assigned to the closest directory group
- [ ] Dry-run does not mutate git state
- [ ] Batching produces correct number of batches for a given size limit
- [ ] Merging combines groups from multiple batch responses
- [ ] Invalid batch responses are skipped without crashing
