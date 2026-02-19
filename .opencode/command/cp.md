---
description: Copy last assistant message to clipboard
allowed-tools: Bash(pbcopy)
---

# Copy to Clipboard

Copy the last assistant message to clipboard using pbcopy.

## Instructions

1. **Extract the last assistant message** from the conversation
   - Get the complete text of your most recent response
   - Include all content (code blocks, lists, formatting)
   - Do NOT include the user's messages

2. **Copy to clipboard**
   - Use: `echo "<message>" | pbcopy`
   - Or: `pbcopy` with piped input for multiline content
   - Prefer heredoc for complex/multiline messages:
   ```bash
   pbcopy <<'EOF'
   [full message content]
   EOF
   ```

3. **Confirm to user**
   - Message: "Last message copied to clipboard"
   - Show character count if helpful

## Notes

- Only copy YOUR last message, not the user's
- Preserve all formatting (markdown, code blocks, etc.)
- Use heredoc with `'EOF'` to prevent variable expansion
