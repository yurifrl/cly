---
name: draft-state
description: Update draft STATE.md - mark drafts as applied/archived
category: Design
tags: [draft, state, tracking]
---

Maintain `.agents/contexts/STATE.md` tracking draft lifecycle.

1. Read `.agents/contexts/STATE.md`
2. Read `.agents/drafts/` to check which drafts exist on disk
3. Ask user which draft(s) to update and to what status (applied/archived)
4. Move entries between Active → Applied → Archived tables
5. Add date and notes to moved entries

**Statuses**: `active` → `applied` (implemented) → `archived` (no longer relevant)

If a draft file exists on disk but not in STATE.md, add it as `active`.
If a STATE.md entry has no matching file, mark it `archived (file removed)`.
