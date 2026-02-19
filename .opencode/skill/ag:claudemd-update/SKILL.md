---
name: ag:claudemd-update
description: Updates CLAUDE.md based on recent conversation history. This skill should be used when the user asks to update their CLAUDE.md, sync learnings from conversations, or review what was discussed recently.
---

# CLAUDE.md Updater

## Extract Conversations

```bash
scripts/extract_conversations.py --project "$(pwd)" --days 1
```

Options: `--days N`, `--limit N`, `--json`

## Workflow

1. **Extract conversations** - Execute the script immediately:
   ```bash
   scripts/extract_conversations.py --project "$(pwd)" --days 1 --limit 15
   ```

2. **Extract final state** - Analyze conversations for the end result:
   - Final decisions and patterns that emerged
   - Current conventions and rules (not how they were discussed)
   - Workflow steps as they should be done now
   - Configuration patterns that were settled on

   **CRITICAL**: Focus on the final product, not the discussion history. Skip:
   - Intermediate steps or abandoned approaches
   - Debugging sessions (unless general pattern)
   - Version-specific details
   - One-time fixes

3. **Present clean documentation** - Show numbered list of rules/patterns to add:
   ```
   Patterns to document:
   1. [Final pattern/rule as it should be done]
   2. [Configuration pattern with current values]
   3. [Workflow step in present tense]
   ...

   Which should I document? (numbers, ranges, or "all")
   ```

4. **Apply updates** - After selection, write clean documentation focused on current state
