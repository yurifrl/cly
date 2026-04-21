# cleanup-github

GitHub account audit and cleanup skill for Claude Code.

## Project Type

Claude Code skill - installed via the trousse plugin

## Origin

Created from a real GitHub cleanup session on 2025-12-28. The session started with investigating failing GitHub Actions and evolved into a comprehensive account audit.

## Key Design Decisions

### Progressive Workflow
Never auto-delete. Always: audit → present → approve → execute.

### "What Did We Miss?" is Mandatory
Phase 5 is not optional. The most valuable findings came from prompting "what did we miss?" - these are now encoded in the audit checklist.

### CodeQL is NOT a Workflow
Critical insight: CodeQL "default setup" is configured via GitHub Security settings, not workflow files. The API is `code-scanning/default-setup`, not `workflows`. Many tools (and Claude) confuse this.

### Secrets Require Cross-Reference
Just listing secrets isn't enough. Must cross-reference with workflow files to prove they're orphaned. And even then, require user approval - secrets might be used by external services.

### Check Local Clones Before Deleting
Deleting a remote repo without checking for local clones creates confusing orphaned git state. Always check `~/Repos` first.

## File Structure

```
cleanup-github/
├── .bon/                # Work tracking (prefix: cg)
├── CLAUDE.md            # This file
├── SKILL.md             # Main skill (<500 lines)
└── references/
    ├── gh-cli-patterns.md      # Complete gh CLI reference
    ├── audit-checklist.md      # "What did we miss?" comprehensive list
    └── cleanup-operations.md   # Safe patterns for destructive ops
```

## Development

### Testing
Run "audit my GitHub" or "clean up my GitHub" in a fresh Claude Code session to test the skill.

### Line Limits
Keep SKILL.md under 500 lines. Move detailed content to references/.

### Iteration
After each real usage, capture gaps and refine the audit checklist.

## Session Learnings Encoded

From the original session:

1. **Failing workflows** - CodeQL was misconfigured for JS/TS but repo had Apps Script (.gs)
2. **Stale forks** - deleted forks with 0 ahead, many behind
3. **Skills fork** - same pattern, also deleted
4. **Orphaned secrets** - 3 secrets no longer used by any workflow - deleted
5. **Local clone check** - verified no local clone before deletion

The "what did we miss?" prompt surfaced: orphaned secrets, Dependabot configs, other stale forks, CodeQL state discrepancy, local clones.
