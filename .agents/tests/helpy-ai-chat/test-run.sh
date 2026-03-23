#!/bin/bash
set -euo pipefail

# Helpy AI Chat — E2E test runner
# Requires: go, vhs, cly built in dist/, ANTHROPIC_API_KEY

cd "$(dirname "$0")/../../.."
ROOT="$PWD"

echo "=== Step 1: Unit tests (frontmatter + llm) ==="
go test ./modules/helpy/... -run TestParseFrontmatter -v -count=1
go test ./pkg/llm/... -v -count=1
echo "✅ Unit tests passed"
echo ""

echo "=== Step 2: Build cly ==="
go build -o dist/cly .
echo "✅ Build succeeded"
echo ""

export PATH="$ROOT/dist:$PATH"

echo "=== Step 3: VHS — Docs picker with frontmatter ==="
vhs .agents/tests/helpy-ai-chat/docs-picker.tape
echo "✅ Docs picker tape completed → .agents/tests/helpy-ai-chat/docs-picker.gif"
echo ""

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo "⚠️  ANTHROPIC_API_KEY not set — skipping AI chat tape"
    echo "Set ANTHROPIC_API_KEY to run the full integration test"
    exit 0
fi

echo "=== Step 4: VHS — AI chat with streaming ==="
vhs .agents/tests/helpy-ai-chat/ai-chat.tape
echo "✅ AI chat tape completed → .agents/tests/helpy-ai-chat/ai-chat.gif"
echo ""

echo "=== All tests passed ✅ ==="
