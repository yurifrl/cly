# GitHub Cleanup Audit Checklist

Comprehensive "what did we miss?" checklist. Run through this BEFORE presenting final findings.

## Core Audit (Always Check)

### Failing Workflows
- [ ] List all repos with workflows
- [ ] Check recent run status for failures
- [ ] Check CodeQL default setup (NOT a workflow file!)
- [ ] Verify language configuration matches repo content

### Stale Forks
- [ ] List ALL forks, not just the ones with failures
- [ ] Check ahead_by AND behind_by for each fork
- [ ] Flag forks with 0 ahead (no custom changes)
- [ ] Verify no local clones before recommending deletion

### Orphaned Secrets
- [ ] List secrets per repo
- [ ] Cross-reference with workflow files
- [ ] Check environment secrets (different API)
- [ ] Present for user review (never auto-delete)

### Security Configuration
- [ ] Check Dependabot configuration
- [ ] Check vulnerability alerts status
- [ ] Check code scanning (CodeQL) status
- [ ] Review push protection settings

### Dependabot Alert Triage
- [ ] Scan all repos for open Dependabot alerts (filter 403s — those mean Dependabot is disabled, not "3 alerts")
- [ ] For each repo with alerts, audit declared deps vs actual imports
- [ ] Identify unused direct deps whose transitive tree contains the vulnerable package
- [ ] Remove unused deps (preferred — permanent fix)
- [ ] Upgrade lock files for remaining transitive alerts (`uv lock --upgrade` / `npm update`)
- [ ] Check for "imported but undeclared" deps (work today via transitive hoisting, break tomorrow)
- [ ] Skip forks of upstream code (not your deps to manage)

### General Dependency Hygiene (all local repos)
- [ ] Find all repos with pyproject.toml or package.json in ~/Repos
- [ ] For each repo, compare declared deps vs actual imports in source
- [ ] Identify declared-but-never-imported deps (removal candidates)
- [ ] Check for "runtime engine" exceptions before removing (openpyxl, lxml, kaleido, pytest-asyncio)
- [ ] Identify imported-but-not-declared deps (fragile transitives to promote)
- [ ] Check for dead imports (imported but variable never used in code)
- [ ] Skip CLI-only dev tools (ruff, black, mypy) — they're never imported

## Often Forgotten

### Local Development
- [ ] **Local clones with stale remotes** - Check ~/Repos for clones of repos being deleted
- [ ] Verify remote URLs won't break after deletion
- [ ] Check for uncommitted work in local clones

### GitHub Features
- [ ] **GitHub Apps installations** - Settings > Applications > Installed GitHub Apps
- [ ] **OAuth app authorizations** - Settings > Applications > Authorized OAuth Apps
- [ ] **Deploy keys** per repository
- [ ] **SSH keys** (global account level)
- [ ] **Webhooks** on repositories
- [ ] **Collaborators** on personal repos
- [ ] **Fine-grained PATs** with excessive scope

### Repository Settings
- [ ] **Archived repos** still triggering actions
- [ ] **Branch protection rules** on repos being modified
- [ ] **Required status checks** that might break

## Security Posture Review

### Visibility
- [ ] Private repos that could be public (open source candidates)
- [ ] Public repos that should be private (accidental exposure)
- [ ] Repos with sensitive content in history

### Security Features
- [ ] Repos with secret scanning disabled
- [ ] Repos with push protection disabled
- [ ] Repos without branch protection on main
- [ ] Repos with force push allowed to main

## Organization Context (if applicable)

- [ ] Stale organization memberships
- [ ] Organization-level secrets
- [ ] Pending team invitations
- [ ] Outside collaborators
- [ ] Organization apps vs personal apps

## Maintenance Signals

### Activity Patterns
- [ ] Repos with no activity >1 year (archive candidates)
- [ ] Repos with only Dependabot commits (abandoned but maintained)
- [ ] Repos with failing CI for >30 days (broken projects)

### Resource Usage
- [ ] GitHub Actions minutes usage
- [ ] Storage usage (LFS, packages)
- [ ] Workflow runs that could be optimized

## Quick Sweep Commands

### Check local clones for stale remotes
```bash
find ~/Repos -maxdepth 2 -name ".git" -type d 2>/dev/null | while read gitdir; do
  repo=$(dirname "$gitdir")
  remote=$(git -C "$repo" remote get-url origin 2>/dev/null)
  echo "$repo: $remote"
done
```

### List GitHub Apps
```bash
gh api user/installations --jq '.installations[].app_slug'
```

### Check for archived repos
```bash
gh repo list USERNAME --json name,isArchived --jq '.[] | select(.isArchived) | .name'
```

### Find repos with no recent activity
```bash
gh repo list USERNAME --json name,pushedAt --jq '.[] | select(.pushedAt < "2024-01-01") | .name'
```

## Verification After Cleanup

- [ ] Confirm deleted repos are gone: `gh repo view USERNAME/REPO` should fail
- [ ] Confirm deleted secrets are removed: re-list secrets
- [ ] Confirm workflows are disabled: check workflow status
- [ ] Verify no broken local remotes: `git fetch` in local clones
- [ ] Test any integrations that might depend on deleted repos
