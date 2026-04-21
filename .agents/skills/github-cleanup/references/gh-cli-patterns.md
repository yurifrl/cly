# gh CLI Patterns for GitHub Cleanup

Complete reference for gh CLI commands used in GitHub account audits.

## Authentication

```bash
# Check auth status
gh auth status

# Get current username
gh api user --jq '.login'

# Add delete_repo scope (required for repo deletion)
gh auth refresh -h github.com -s delete_repo
```

## Repository Listing

```bash
# List all repos with metadata
gh repo list USERNAME --limit 100 --json name,pushedAt,isArchived,visibility,isFork

# List only forks with parent info
gh repo list USERNAME --fork --json name,parent \
  --jq '.[] | "\(.name) (fork of \(.parent.nameWithOwner // "unknown"))"'

# Count repos
gh repo list USERNAME --limit 1000 --json name --jq 'length'

# Filter by visibility
gh repo list USERNAME --visibility private --json name --jq '.[].name'
```

## Workflows

```bash
# List workflows in a repo
gh workflow list --repo USERNAME/REPO

# List all workflows including disabled
gh workflow list --repo USERNAME/REPO --all

# Check recent runs
gh run list --repo USERNAME/REPO --limit 5 --json status,conclusion,name,createdAt

# Find failing runs
gh run list --repo USERNAME/REPO --json conclusion --jq '[.[] | select(.conclusion == "failure")] | length'

# Disable a workflow
gh workflow disable "Workflow Name" --repo USERNAME/REPO

# Enable a workflow
gh workflow enable "Workflow Name" --repo USERNAME/REPO
```

## Code Scanning (CodeQL)

**Important:** CodeQL "default setup" is NOT a workflow file. It's configured via GitHub Security settings and has a separate API.

```bash
# Check CodeQL default setup status
gh api repos/USERNAME/REPO/code-scanning/default-setup --jq '.state'
# Returns: "configured" or "not-configured"

# Get full CodeQL config
gh api repos/USERNAME/REPO/code-scanning/default-setup \
  --jq '{state: .state, languages: .languages, schedule: .schedule}'

# Disable CodeQL default setup
gh api repos/USERNAME/REPO/code-scanning/default-setup -X PATCH -f state=not-configured

# Enable CodeQL default setup
gh api repos/USERNAME/REPO/code-scanning/default-setup -X PATCH \
  -f state=configured \
  -f languages='["javascript","python"]'
```

## Fork Comparison

```bash
# Compare fork to upstream (get ahead/behind counts)
gh api repos/USERNAME/REPO/compare/UPSTREAM_OWNER:main...USERNAME:main \
  --jq '{ahead: .ahead_by, behind: .behind_by, status: .status}'

# Check if repo is a fork and get parent
gh api repos/USERNAME/REPO --jq '{isFork: .fork, parent: .parent.full_name}'

# Get fork's source repo
gh api repos/USERNAME/REPO --jq '.source.full_name'
```

## Secrets

```bash
# List repository secrets (names only, values not accessible)
gh api repos/USERNAME/REPO/actions/secrets --jq '.secrets[].name'

# Count secrets
gh api repos/USERNAME/REPO/actions/secrets --jq '.total_count'

# Delete a secret
gh api repos/USERNAME/REPO/actions/secrets/SECRET_NAME -X DELETE

# Check environment secrets (different endpoint)
gh api repos/USERNAME/REPO/environments --jq '.environments[].name'
gh api repos/USERNAME/REPO/environments/ENV_NAME/secrets --jq '.secrets[].name'
```

## Repository Deletion

```bash
# Delete a repository (requires delete_repo scope)
gh repo delete USERNAME/REPO --yes

# If scope missing, add it first:
gh auth refresh -h github.com -s delete_repo
```

## Dependabot

```bash
# Check if dependabot.yml exists
gh api repos/USERNAME/REPO/contents/.github/dependabot.yml --jq '.name' 2>/dev/null \
  && echo "Dependabot configured"

# Get dependabot.yml content
gh api repos/USERNAME/REPO/contents/.github/dependabot.yml --jq '.content' | base64 -d

# Check vulnerability alerts status
gh api repos/USERNAME/REPO/vulnerability-alerts 2>/dev/null \
  && echo "Vulnerability alerts enabled"
```

## Workflow File Content

```bash
# List workflow files
gh api repos/USERNAME/REPO/contents/.github/workflows --jq '.[].name'

# Get workflow file content
gh api repos/USERNAME/REPO/contents/.github/workflows/FILE.yml --jq '.content' | base64 -d

# Search for secret references in workflow
gh api repos/USERNAME/REPO/contents/.github/workflows/FILE.yml --jq '.content' | base64 -d \
  | grep -o 'secrets\.[A-Z_]*' | sort -u
```

## Batch Operations

```bash
# Iterate over all repos (bash required for loops)
/bin/bash -c 'for repo in $(gh repo list USERNAME --limit 100 --json name --jq ".[].name"); do
  echo "Processing $repo..."
  # your command here
done'

# Iterate over forks only
/bin/bash -c 'for repo in $(gh repo list USERNAME --fork --json name --jq ".[].name"); do
  echo "Fork: $repo"
done'
```

## jq Patterns

```bash
# Filter by condition
gh repo list USERNAME --json name,isFork --jq '.[] | select(.isFork == true) | .name'

# Format output
gh repo list USERNAME --json name,pushedAt --jq '.[] | "\(.name): \(.pushedAt)"'

# Count items
gh api repos/USERNAME/REPO/actions/secrets --jq '.secrets | length'

# Extract single value from array (bd show pattern)
gh api repos/USERNAME/REPO --jq '.full_name'
```

## Error Handling

```bash
# Check if API call succeeded
gh api repos/USERNAME/REPO 2>/dev/null && echo "Exists" || echo "Not found"

# Suppress errors for optional checks
gh api repos/USERNAME/REPO/code-scanning/default-setup 2>/dev/null || echo "Not configured"

# Check exit code
gh repo view USERNAME/REPO 2>&1 | grep -q "not found" && echo "Confirmed deleted"
```
