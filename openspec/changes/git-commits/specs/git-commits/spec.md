## ADDED Requirements

### Requirement: Analyze git changeset
The system SHALL parse git diff output to produce a structured changeset of file-level changes with path, status (A/M/D/R), old path (renames), and hunk metadata.

#### Scenario: Staged changes analyzed by default
- **WHEN** user runs `cly git-commits` with staged changes present
- **THEN** the system parses `git diff --cached` and produces a changeset with one entry per changed file

#### Scenario: All changes with --all flag
- **WHEN** user runs `cly git-commits --all`
- **THEN** the system stages all changes (including untracked files via `git add .`) before analyzing

#### Scenario: No changes detected
- **WHEN** user runs `cly git-commits` with no staged changes
- **THEN** the system prints "No staged changes found" and exits with error

#### Scenario: Renamed files detected
- **WHEN** a file has been renamed (git reports R status)
- **THEN** the changeset entry includes both `path` (new) and `oldPath` (original)

### Requirement: Batch large changesets
The system SHALL split changeset analysis text into batches of at most 40,000 characters (configurable) to fit within LLM context windows.

#### Scenario: Small changeset fits in one batch
- **WHEN** total analysis text is under 40,000 characters
- **THEN** exactly one batch is produced containing all files

#### Scenario: Large changeset split into multiple batches
- **WHEN** total analysis text exceeds 40,000 characters
- **THEN** files are packed greedily into multiple batches, each under the limit, splitting only on file boundaries

#### Scenario: Custom batch size via config
- **WHEN** `modules.git-commits.batch-size` is set to 20000 in config
- **THEN** batches are capped at 20,000 characters

### Requirement: Plan commit split via LLM
The system SHALL send each batch to the configured LLM provider and receive a JSON plan grouping files into logical commits with conventional commit messages.

#### Scenario: Single batch planning
- **WHEN** one batch is sent to the LLM
- **THEN** the response contains a JSON object with `groups` array, each group having `title`, `type`, `summary`, and `items` (file list)

#### Scenario: Parallel multi-batch planning
- **WHEN** multiple batches exist
- **THEN** all batches are sent to the LLM concurrently and results are merged into a single plan

#### Scenario: Batch failure tolerance
- **WHEN** one batch request fails (timeout, invalid JSON) but others succeed
- **THEN** surviving batch results are merged and uncovered files are auto-assigned

#### Scenario: Total planning failure fallback
- **WHEN** all batch requests fail
- **THEN** the system falls back to a single commit with an AI-generated conventional commit message

### Requirement: Validate and auto-heal plan
The system SHALL validate the LLM plan against the actual changeset and auto-heal recoverable issues.

#### Scenario: Duplicate files across groups
- **WHEN** the LLM places the same file in multiple groups
- **THEN** the file is kept in its first occurrence and removed from later groups; empty groups are dropped

#### Scenario: Uncovered files
- **WHEN** the LLM plan omits files that are in the changeset
- **THEN** each uncovered file is assigned to the group whose existing files share the longest directory prefix

#### Scenario: Empty plan after healing
- **WHEN** the plan has zero groups after healing
- **THEN** the system rejects the plan and falls back to single-commit mode

### Requirement: Preview commit plan
The system SHALL display the commit plan to the user in a readable format before execution.

#### Scenario: Default interactive preview
- **WHEN** user runs `cly git-commits` (no flags)
- **THEN** the plan is printed with numbered commits, each showing title, summary, and file list, followed by a `Execute? [Y/n]` prompt

#### Scenario: Dry run mode
- **WHEN** user runs `cly git-commits --dry-run`
- **THEN** the plan is printed and the command exits without any git mutations

#### Scenario: JSON output mode
- **WHEN** user runs `cly git-commits --json`
- **THEN** the raw JSON plan is printed to stdout and the command exits without git mutations

#### Scenario: Auto-confirm mode
- **WHEN** user runs `cly git-commits --yes`
- **THEN** the plan is printed and executed immediately without confirmation prompt

### Requirement: Execute commit plan
The system SHALL create one git commit per group in the plan, staging only the group's files for each commit.

#### Scenario: Multi-commit execution
- **WHEN** the plan has 3 groups and user confirms
- **THEN** 3 separate commits are created in order, each containing only its group's files with the group's title as commit message

#### Scenario: Added files staged correctly
- **WHEN** a group contains an added file
- **THEN** `git add <path>` is used to stage it before committing

#### Scenario: Deleted files staged correctly
- **WHEN** a group contains a deleted file
- **THEN** `git rm --cached <path>` is used to stage the deletion before committing

#### Scenario: Renamed files staged correctly
- **WHEN** a group contains a renamed file
- **THEN** `git rm --cached <oldPath>` and `git add <newPath>` are used to stage the rename

#### Scenario: No-verify flag bypasses hooks
- **WHEN** user runs `cly git-commits --no-verify`
- **THEN** `--no-verify` is passed to each `git commit` call

### Requirement: Rollback on execution failure
The system SHALL restore the original git state if commit execution fails partway through.

#### Scenario: Staging failure triggers rollback
- **WHEN** `git add` fails for a file during execution
- **THEN** the system runs `git reset --soft <originalHead>`, then `git reset`, then `git apply --cached < savedDiff>` to restore the original state

#### Scenario: Rollback failure reports original HEAD
- **WHEN** the rollback procedure itself fails
- **THEN** the system prints the original HEAD SHA so the user can manually recover with `git reset --soft <sha>`

### Requirement: Command registration and aliases
The system SHALL register as `cly git-commits` with alias `gc`.

#### Scenario: Full command name
- **WHEN** user runs `cly git-commits`
- **THEN** the command executes the split-commit workflow

#### Scenario: Alias
- **WHEN** user runs `cly gc`
- **THEN** the command executes identically to `cly git-commits`

### Requirement: Performance target
The system SHALL complete the full workflow (analyze → plan → execute) in under 5 seconds for changesets with fewer than 20 files on a typical network connection.

#### Scenario: Small changeset performance
- **WHEN** user runs `cly git-commits` with 10 staged files
- **THEN** the total wall-clock time from invocation to final commit is under 5 seconds

#### Scenario: Large changeset bounded by LLM latency
- **WHEN** user runs `cly git-commits` with 60 staged files producing 3 batches
- **THEN** batches are processed in parallel, keeping total time close to single-batch latency plus overhead
