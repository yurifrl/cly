# GitHub Cleanup Skill

Progressive audit and cleanup of GitHub accounts — stale forks, orphaned secrets, failing workflows, security configs.

## What This Skill Does

Audits a GitHub account and presents findings for approval before any destructive actions:
- **Failing workflows** — Including CodeQL misconfiguration
- **Stale forks** — No custom changes, far behind upstream
- **Orphaned secrets** — Not referenced in any workflow
- **Security config** — Dependabot, vulnerability alerts

## Installation

Install via batterie-de-savoir marketplace: `/plugin marketplace add spm1001/batterie-de-savoir` then `/plugin install trousse`

## When Claude Uses This Skill

Activates on:
- "clean up my GitHub", "audit my repos"
- "check for stale forks", "orphaned secrets"
- "GitHub hygiene"

## Workflow

1. **Audit** all categories
2. **Present** consolidated findings
3. **Get approval** via AskUserQuestion
4. **Execute** only approved actions
5. **Verify** cleanup succeeded

## File Structure

```
cleanup-github/
├── SKILL.md                  # Main skill
├── CLAUDE.md                 # Design decisions and origin
└── references/
    ├── gh-cli-patterns.md    # Complete gh CLI reference
    ├── audit-checklist.md    # "What did we miss?" checklist
    └── cleanup-operations.md # Safe patterns for destructive ops
```

## Key Insight: CodeQL

CodeQL "default setup" is configured via GitHub Security settings, NOT workflow files. The API is `code-scanning/default-setup`, not `workflows`. Many tools get this wrong.

## Requirements

- `gh` CLI installed and authenticated
- `delete_repo` scope for fork deletion: `gh auth refresh -s delete_repo`

## Safety

- Never auto-deletes — always requires user approval
- Checks for local clones before deleting remote repos
- Cross-references secrets with workflow files

## License

MIT
