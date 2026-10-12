---
description: List and continue from a saved context
allowed-tools: Bash(fish:*), Read(**)
---

# Continue

List recent contexts and continue from one.

## Instructions

1. **List contexts** from `.agents/contexts/`
   - Run: `fish -c "ls -t .agents/contexts/*.md 2>/dev/null | head -20"`
   - Parse only frontmatter from each file (not full content)
   - Extract: created, project, description

2. **Display list** in format:
```
Recent contexts:
[1] 250130142315-fix-auth-flow - "Fix token refresh logic" (myproject)
[2] 250129103000-add-dark-mode - "Dark mode toggle" (myproject)
...
[q] Cancel
```

3. **Ask user**: Which context to continue from?

4. **Load selected context**:
   - Read the full file
   - Display content to give LLM full context to continue
