---
name: draft
description: Create draft from user input
category: Design
tags: [draft, design, planning]
---

Create `.agents/drafts/[name].md` from user's text.

1. Read `.agents/contexts/STATE.md` to see current draft state
2. Generate filename from the text (kebab-case)
3. Organize ideas using drafts skill
4. Create the file in `.agents/drafts/` (see skill for location rules)
5. Add entry to `.agents/contexts/STATE.md` Active Drafts table with date and status `active`

**CRITICAL**: Draft = architecture, not full implementation tutorials.
**CRITICAL**: Always update STATE.md when creating a draft.
