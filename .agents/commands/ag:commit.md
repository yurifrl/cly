---
description: "Creates well-formatted commits with conventional commit messages, using line-level change grouping"
allowed-tools:
  [
    "Bash(git add:*)",
    "Bash(git status:*)",
    "Bash(git commit:*)",
    "Bash(git diff:*)",
    "Bash(git log:*)",
    "Bash(git branch:*)",
  ]
---

# Claude Command: Commit

Creates atomic, well-formatted commits by intelligently grouping changes at the line level.

## Usage

/commit
/commit --no-verify

## Process

### 1. **Discover Changes**
- Scan all modified/new/deleted files with `git status`
- Extract line-level changes (hunks) from `git diff --unified=0`
- Parse individual change blocks with their file locations

### 2. **Group Changes**
Apply logical heuristics to create commit groups. A group can be:
- An entire file (new/deleted/small changes)
- Specific line ranges within files
- Multiple related files or hunks

**Grouping Heuristics (in priority order):**

**A. Entire File Groups:**
- New files (untracked)
- Deleted files
- File renames/moves
- Small changes (<10 lines total)
- Binary files

**B. Line-Level Groups (for larger changes):**
1. **Functional proximity** - hunks within ~20 lines in same file
2. **Same concern** - multiple hunks in one file with similar patterns:
   - Import statements
   - Same function/class scope
   - Related variable/function names
3. **Cross-file logic** - related changes across files:
   - Same feature/concern (matching file paths)
   - Config + implementation pairs
   - Test + implementation pairs
   - Related configs (e.g., fish config + fish functions)

### 3. **Review Groups**
Present proposed grouping with commit message preview:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Group 1: Session export feature
💬 feat: add session export to claudext wrapper

  • home/.config/fish/functions/claudext.fish
    Lines: 45-67 (export session logic)
    Lines: 123-145 (export flag handling)
  • HELP.md
    Lines: 234-256 (export documentation)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- Show proposed first line (commit message) for each group
- Allow user to: accept, adjust grouping, merge/split groups

### 4. **Commit Each Group**
For each approved group:
1. Stage only the group's specific files/hunks
2. Show full staged diff
3. Generate complete commit message (first line + body)
4. Get approval and commit
5. Reset staging for next group

## Commit Format

`<type>: <description>`

### Types:
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring
- `docs`: Documentation only
- `style`: Formatting/whitespace
- `perf`: Performance improvement
- `test`: Test additions/changes
- `chore`: Build/tools/dependencies
- `config`: Configuration changes

### Rules:
- **Imperative mood** ("add" not "added")
- **First line <72 chars**
- **Atomic commits** (single logical purpose)
- **Problem-focused** - explain what problem it solves, not just what changed
- **No Claude signature** (explicitly forbidden)

### Commit Body (auto-generated):
- What problem this solves
- List of affected files (if multi-file)
- Context for future reference

## Splitting Criteria

Split into separate groups when:
- Different types (feat vs fix vs chore)
- Unrelated concerns or features
- Different functional areas
- Mixed refactoring + new features

## Options

- `--no-verify`: Skip git hooks (Husky, etc.)

## Notes

- **Branch check**: If on `main`, create new branch starting with `yurifrl/<type>/...`
- **Line-level precision**: Changes grouped by logical hunks, not just files
- **Smart grouping**: Related changes stay together even across files
- **Review before commit**: Always show grouping + proposed messages first

## Examples

### Example: Multiple concerns in one file
```
File: config.fish (changes at lines 10-15, 67-89, 120-125)

Group 1: Lines 10-15, 67-89 (session management - related)
💬 feat: add session persistence to fish config

Group 2: Lines 120-125 (brew updates - unrelated)
💬 chore: update brew package list
```

### Example: Cross-file feature
```
Group 1: Feature implementation
💬 feat: add keyboard firmware flash command

  • keyboard/Taskfile.yaml (entire file - new task)
  • HELP.md (lines 45-67 - documentation)
  • home/.config/fish/functions/kb.fish (lines 23-45 - wrapper)
```
