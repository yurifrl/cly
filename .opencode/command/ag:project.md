---
description: Work with .project.md context files
---

Activate the **project** skill.

User request: {{PROMPT}}

This is a catchall command for .project.md operations:
- If project exists → load and update it
- If project doesn't exist → create it
- If context is loose/ambiguous → ask user if they want to use .project.md

Use the `global:project` agent (Task tool, subagent_type="global:project") to handle operations.
