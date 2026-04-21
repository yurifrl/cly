# Cleanup Operations Reference

Safe patterns for destructive operations during GitHub cleanup.

## Safety Principles

1. **Audit first, delete later** - Never delete in the same step as discovery
2. **User approval required** - All destructive operations need explicit approval
3. **Verify after deletion** - Confirm operations completed successfully
4. **Check dependencies first** - Local clones, integrations, downstream users

## Pre-Deletion Checks

### Before Deleting a Fork

```bash
# 1. Check for local clone
ls ~/Repos | grep -i REPO_NAME
find ~/Repos -maxdepth 2 -name ".git" -exec dirname {} \; | xargs -I {} sh -c 'git -C "{}" remote get-url origin 2>/dev/null | grep -qi REPO_NAME && echo "{}"'

# 2. Verify no unique commits (ahead_by should be 0)
gh api repos/USERNAME/REPO/compare/UPSTREAM:main...USERNAME:main --jq '.ahead_by'

# 3. Check for open PRs from the fork
gh pr list --repo UPSTREAM_OWNER/UPSTREAM_REPO --author USERNAME

# 4. Check if fork is referenced in any documentation
# (manual check - search docs for fork URL)
```

### Before Deleting Secrets

```bash
# 1. List all workflow files
gh api repos/USERNAME/REPO/contents/.github/workflows --jq '.[].name'

# 2. Search each workflow for the secret name
/bin/bash -c 'for file in $(gh api repos/USERNAME/REPO/contents/.github/workflows --jq ".[].name"); do
  echo "=== $file ==="
  gh api "repos/USERNAME/REPO/contents/.github/workflows/$file" --jq ".content" | base64 -d | grep -i SECRET_NAME || echo "Not found"
done'

# 3. Check environment secrets (different location)
gh api repos/USERNAME/REPO/environments --jq '.environments[].name' | while read env; do
  gh api "repos/USERNAME/REPO/environments/$env/secrets" --jq '.secrets[].name' | grep -i SECRET_NAME
done

# 4. Consider external services that might use the secret
# (manual check - CI services, deployment targets, etc.)
```

### Before Disabling Workflows

```bash
# 1. Check if workflow is actively used
gh run list --repo USERNAME/REPO --workflow "Workflow Name" --limit 10 --json createdAt,conclusion

# 2. Check if workflow is required for branch protection
gh api repos/USERNAME/REPO/branches/main/protection --jq '.required_status_checks.contexts[]' 2>/dev/null

# 3. Verify you understand what the workflow does
gh api repos/USERNAME/REPO/contents/.github/workflows/WORKFLOW.yml --jq '.content' | base64 -d | head -30
```

## Deletion Commands

### Delete Repository

```bash
# Requires delete_repo scope
gh auth refresh -h github.com -s delete_repo

# Delete with confirmation bypass (--yes)
gh repo delete USERNAME/REPO --yes
```

### Delete Secret

```bash
# Repository secret
gh api repos/USERNAME/REPO/actions/secrets/SECRET_NAME -X DELETE

# Environment secret
gh api repos/USERNAME/REPO/environments/ENV_NAME/secrets/SECRET_NAME -X DELETE
```

### Disable Security Features

```bash
# Disable CodeQL default setup
gh api repos/USERNAME/REPO/code-scanning/default-setup -X PATCH -f state=not-configured

# Disable workflow
gh workflow disable "Workflow Name" --repo USERNAME/REPO
```

## Verification After Deletion

### Confirm Repo Deleted

```bash
# Should return error
gh repo view USERNAME/REPO 2>&1 | grep -q "not found" && echo "Confirmed deleted" || echo "Still exists!"
```

### Confirm Secret Deleted

```bash
# Secret should not appear in list
gh api repos/USERNAME/REPO/actions/secrets --jq '.secrets[].name' | grep -q SECRET_NAME && echo "Still exists!" || echo "Confirmed deleted"
```

### Confirm Workflow Disabled

```bash
# Should show "disabled_manually"
gh workflow list --repo USERNAME/REPO --all | grep "Workflow Name"
```

### Verify Local Clones

```bash
# In local clone directory
cd ~/Repos/LOCAL_CLONE
git fetch  # Should fail if remote deleted
# If this was intentional, remove the remote or delete local clone
```

## Batch Operation Patterns

### Collect, Present, Approve, Execute

```bash
# 1. COLLECT - gather all findings
findings=""
for repo in $(gh repo list USERNAME --fork --json name --jq '.[].name'); do
  ahead=$(gh api "repos/USERNAME/$repo/compare/..." --jq '.ahead_by' 2>/dev/null || echo "unknown")
  if [ "$ahead" = "0" ]; then
    findings="$findings\n$repo (stale fork)"
  fi
done

# 2. PRESENT - show user what will be affected
echo "Found stale forks:"
echo -e "$findings"

# 3. APPROVE - get explicit confirmation
# Use AskUserQuestion tool

# 4. EXECUTE - only after approval
# Run deletion commands
```

### Never Auto-Execute

```bash
# BAD - auto-deletes without approval
for repo in $stale_forks; do
  gh repo delete "USERNAME/$repo" --yes
done

# GOOD - presents for approval first
echo "Stale forks to delete: $stale_forks"
# Wait for user confirmation via AskUserQuestion
```

## Error Handling

### Scope Errors

```bash
# If you see "Must have admin rights" or scope errors
gh auth refresh -h github.com -s delete_repo,security_events

# Verify scopes
gh auth status
```

### Rate Limiting

```bash
# Check rate limit status
gh api rate_limit --jq '.rate'

# If rate limited, wait or reduce batch size
```

### Partial Failures

```bash
# Log each operation result
for repo in $repos_to_delete; do
  if gh repo delete "USERNAME/$repo" --yes 2>/dev/null; then
    echo "Deleted: $repo"
  else
    echo "FAILED: $repo"
  fi
done
```

## Rollback Considerations

### Deleted Repos
- **Cannot be restored** - deletion is permanent
- Fork history remains in upstream if it was a fork
- Consider archiving instead of deleting if uncertain

### Deleted Secrets
- **Cannot be restored** - you need the original value
- Document what secrets existed before deletion
- Ensure you have values stored securely elsewhere

### Disabled Workflows
- **Can be re-enabled** - `gh workflow enable "Name" --repo ...`
- Workflow files still exist in repo
- Just need to re-enable via API or UI
