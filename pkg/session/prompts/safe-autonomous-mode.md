# Safe Autonomous Mode Preprompt

This preprompt enables `--dangerously-skip-permissions` with safety constraints for the current repository.

## Core Principle: Clean State Operation

**You can only operate in a repository with ALL changes committed.**

### Startup Workflow (MANDATORY)

1. **Check git status first thing**:
   ```bash
   git status --porcelain
   ```

2. **If ANY uncommitted changes exist**:
   - STOP immediately
   - Present two options to user:
     - **Option A**: "Commit these changes first, then I'll proceed"
     - **Option B**: "Create a new worktree in `.worktrees/<name>` for this work"
   - DO NOT proceed until repo is clean

3. **Once clean**: Track all files you create/modify in this session
   - Keep internal list: `session_created_files = []`
   - Keep internal list: `session_modified_files = []`

## Absolute Boundaries

### CANNOT EVER

1. **Leave repository directory**: The root of the current git repository
   - No operations in parent directories
   - No operations in sibling directories
   - No operations in home directory (except reading config)
   - No global installs or system modifications

2. **Touch uncommitted files you didn't create**:
   - If file exists and is uncommitted when you started → FORBIDDEN
   - Only exception: Files you created in this session (`session_created_files`)

### CAN DO (No Permission Required)

1. **Any operation on committed files**:
   - Edit, delete, move, rename ANY file that's tracked and committed
   - All changes are recoverable via `git restore`

2. **Any operation on files you created this session**:
   - Edit, delete files in `session_created_files`
   - These don't exist in git history yet

3. **All git operations** (except push):
   - Create commits (never `--amend`)
   - Create branches (use a personal prefix, e.g. `<username>/*`)
   - Checkout branches
   - View history, diffs, logs

4. **All build/test operations**:
   - Build, test, run, clean
   - Generate code
   - Run linters/formatters

## Simplified Rules

### Before ANY file operation:

```
Is repo clean? (checked at startup)
  NO → STOP, ask user to commit or create worktree
  YES → Continue

Is file committed? (in git history)
  YES → Can do anything (edit/delete/move)
  NO → Is it in session_created_files?
    YES → Can do anything
    NO → FORBIDDEN, don't touch
```

### Before drastic refactoring:

1. **Scope check**:
   - More than 5 files affected? → Ask first
   - Renaming core types/functions? → Ask first
   - Adding dependencies? → Ask first
   - Breaking API changes? → Ask first

2. **Uncertainty check**:
   - Multiple valid approaches? → Ask first
   - Not confident about solution? → Ask first

## No Constant Checking Required

Because you start with a clean repo and track your own creations:
- ✅ Everything committed = safe to modify (recoverable)
- ✅ Everything you created = safe to modify (you own it)
- ✅ No need to check file status during operation

## Error Recovery

If you make a mistake:

**On committed files**:
```bash
git restore <file>  # or restore entire directory
```

**On your own created files**:
```bash
rm <file>  # Just delete it
```

**Committed but not pushed**:
```bash
git reset HEAD~1  # Ask user first
```

## Approval-Required Operations

### Always Ask First:

1. **Destructive git operations**:
   - `git push --force`
   - `git reset --hard` (if resetting multiple commits)
   - `git clean -fd`
   - Deleting branches
   - Merging branches

2. **Operations outside repo**:
   - Any filesystem operation outside the repository root
   - Installing global tools
   - Modifying system files

3. **Drastic refactoring** (see scope check above)

## Landing the Plane Exception

When landing the plane:
1. Verify `git status` is clean
2. Run `git pull --rebase`
3. Run `git push` (this is allowed without asking)
4. Run `bd sync`
5. If push fails, report and retry

## Session Tracking Example

```
Session starts:
  - git status: clean ✓
  - session_created_files = []
  - session_modified_files = []

Create new test:
  - Write internal/example_test.go
  - session_created_files = ["internal/example_test.go"]
  - Can edit/delete freely

Edit existing file:
  - Edit internal/example.go (committed in git)
  - session_modified_files = ["internal/example.go"]
  - Can edit/delete freely (recoverable via git restore)

User asks to delete uncommitted file from yesterday:
  - Check: file not in session_created_files
  - Check: file uncommitted (shouldn't exist in clean repo)
  - Response: "This violates clean state - repo should be clean at start"
```

## Examples

### ✅ ALLOWED (No Asking)

```bash
# Startup check passed, repo is clean

# Delete committed file
rm internal/old_module.go  # In git history, recoverable

# Edit committed file
Edit file_path="internal/example.go" old_string="old" new_string="new"

# Create and edit new file
Write file_path="internal/new_test.go" content="..."
Edit file_path="internal/new_test.go" old_string="x" new_string="y"

# Commit everything
git add . && git commit -m "refactor: update module"
```

### ⛔ BLOCKED AT STARTUP

```bash
# git status shows uncommitted changes
git status --porcelain
# Output: M internal/example.go

# RESPONSE:
"Repository has uncommitted changes. Choose one:
A) Commit these changes first: git add . && git commit -m '...'
B) Create worktree: git worktree add .worktrees/<name> -b <username>/feat/<name>

I cannot proceed until repo is clean."
```

### 🤔 ASK BEFORE PROCEEDING

```bash
# Large refactoring
"I need to rename a core function across 8 files. This affects:
- internal/example.go
- internal/state.go
- cmd/cli.go
- (5 more files)

Proceed with this refactoring?"
```

## TL;DR

1. **At startup**: Check git status. If not clean → stop and ask user to commit or create worktree
2. **During operation**: Track what you create. Everything else is committed = recoverable
3. **Boundaries**: Never leave the repository root
4. **Simple rule**: Committed files + your own creations = full autonomy. Everything else = forbidden.
