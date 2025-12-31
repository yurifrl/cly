---
description: Create a release with version bump and changelog
agent: build
---

Create a release by analyzing changes and creating a plan for user approval.

## Analysis Phase


NOTE: IN SHORT .ai is not for that, dont nest things that much anyway)
NOTE: run this mostly in memoty, only artifact is the changed docs)

1. Run these commands to understand current state:
   - `!git status` - see uncommitted changes
   - `!git diff` - analyze what changed
   - `!cat VERSION` - get current version
   - `!git log --oneline -20` - recent commits for changelog

2. **Analyze documentation impact (create plan, don't auto-update):**

   Check staged changes and FLAG what docs might need updating:
   - New modules added? → suggest .ai/context/modules/<name>.md (NOTE: wht is this? where did u that from? thes no ai moduesl context modues, this is far from code, makes no sesnse here, .ai is for ONLY things that we know ai looks /commands /skills /agents
   - Pattern files changed (pkg/*)? → suggest reviewing .ai/context/patterns/ (not the palce and the ideai is bad)
   - Config changed? → suggest reviewing .ai/context/CONFIG.md (note: shit)
   - New interfaces/major refactor? → suggest new ADR in .ai/context/adr/ (note: shit)

   SHOW PLAN of doc (note: AND COMMIT DOC AND COMMIT PLAN, COMMITS MULTIPLE I EXPECT MULTIPLES WHNE MAKE SENSE) updates (don't create files automatically).

3. Analyze the changes and create a plan showing:
   - List of files that will be committed
   - Documentation updates needed (if any)
   - Suggest bump type (major/minor/patch) based on changes
   - Show old version -> new version (ok, but dont focus on this)
   - Preview the changelog entries

4. Ask user:
   - Which bump type to use (major/minor/patch)
   - Confirm to proceed

## Execution Phase (after approval)

1. Commit all changes: `git add . && git commit -m "chore: prepare release"` (note: single commit sux breck things down"
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
