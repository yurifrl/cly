---
description: Create a release with version bump and changelog
agent: build
---

Create a release by analyzing changes and creating a plan for user approval.

## Analysis Phase

1. Run these commands to understand current state:
   - `!git status` - see uncommitted changes
   - `!git diff` - analyze what changed
   - `!cat VERSION` - get current version
   - `!git log --oneline -20` - recent commits for changelog

2. Analyze the changes and create a plan showing:
   - List of files that will be committed
   - Suggest bump type (major/minor/patch) based on changes
   - Show old version -> new version
   - Preview the changelog entries

3. Ask user:
   - Which bump type to use (major/minor/patch)
   - Confirm to proceed

## Execution Phase (after approval)

1. Commit all changes: `git add . && git commit -m "chore: prepare release"`
2. Bump VERSION file based on user's choice
3. Generate CHANGELOG.md entry with format:
   ```
   ## [X.X.X] - YYYY-MM-DD
   
   - commit message 1
   - commit message 2
   ```
4. Prepend to existing CHANGELOG.md (or create if doesn't exist)
5. Commit version bump: `git add VERSION CHANGELOG.md && git commit -m "chore: bump version to X.X.X"`
6. Push: `git push`

Show each step as you execute it.
