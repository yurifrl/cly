# Test: Helpy AI Chat

## Context
Validates the helpy AI chat feature end-to-end: frontmatter parsing, docs picker with descriptions, AI chat panel toggle, and real AI streaming response.

## Workflow

### 1. Verify frontmatter parsing (unit — no TUI)
```bash
cd /Users/yuri/Workdir/Yuri/cly
go test ./modules/helpy/... -run TestParseFrontmatter -v
```

### 2. Verify pkg/llm client creation
```bash
go test ./pkg/llm/... -v
```

### 3. TUI: Docs picker shows descriptions (VHS)
```bash
vhs .agents/tests/helpy-ai-chat/docs-picker.tape
```
Opens helpy docs picker, verifies docs load with descriptions from frontmatter.

### 4. TUI: AI chat panel opens and streams (VHS)
```bash
vhs .agents/tests/helpy-ai-chat/ai-chat.tape
```
Opens a doc, toggles AI chat with Ctrl+A, types a question, verifies streaming response appears.

## Assertions

| Step | What to check | How |
|------|--------------|-----|
| 1 | Frontmatter parser works | `go test` passes for all 6 frontmatter tests |
| 2 | LLM client creates for both providers | `go test` passes for all 8 llm tests |
| 3 | Docs picker shows descriptions | VHS output gif shows description text under doc names |
| 4 | Chat panel opens on Ctrl+A | VHS output gif shows chat textarea appearing |
| 4 | AI responds to question | VHS output gif shows streamed response text |

## Cleanup
VHS tapes produce output gifs in `.agents/tests/helpy-ai-chat/`. Delete them when done.
