---
project: false
gitignored: false
---

Run CodeRabbit review and address any fixes needed.

Defaults to use --base main to compare against the main branch if currently in a branch.

The user may optionally specify "uncommitted" or "committed" to explicitly choose the review type.

Steps:
1. Check git status to determine if there are uncommitted changes
2. If user specified a type (uncommitted/committed), use that
3. Otherwise, automatically decide:
   - If there are uncommitted changes: use --type uncommitted
   - If no uncommitted changes: run without --type flag (reviews committed changes)
4. If not command is found use --base main if in a branch
5. Run: coderabbit review --plain [--type uncommitted if applicable]
6. Review the output for any issues, suggestions, or fixes
7. Address each fix by making the necessary code changes
8. Verify all changes are applied correctly
