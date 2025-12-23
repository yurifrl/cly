#!/usr/bin/env bash
# Sync .ai/ to AI tool directories

set -e

echo "🔄 Syncing .ai/ to AI tools..."

# Claude Code
if [ -d ".claude" ]; then
    rsync -av --delete .ai/ .claude/
    echo "✓ Synced to .claude/"
fi

# Cursor IDE (future)
# if [ -d ".cursor" ]; then
#     mkdir -p .cursor/rules/
#     cp .ai/docs/*.md .cursor/rules/ 2>/dev/null || true
#     echo "✓ Synced to .cursor/"
# fi

# Crush AI (future)
# if [ -d ".crush" ]; then
#     rsync -av --delete .ai/ .crush/
#     echo "✓ Synced to .crush/"
# fi

# OpenCode (future)
# if [ -d ".opencode" ]; then
#     rsync -av --delete .ai/ .opencode/
#     echo "✓ Synced to .opencode/"
# fi

echo "✅ Sync complete"
