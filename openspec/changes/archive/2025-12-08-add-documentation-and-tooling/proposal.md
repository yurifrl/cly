# Change: Add Documentation and Developer Tooling

## Why
CLY needs user-facing documentation (README) and developer tooling to maintain quality and velocity. The README makes the project discoverable and usable. The Claude Code skill automates module creation following established patterns, ensuring consistency and reducing manual work. Updated module template reflects current architecture (48 demo modules + utilities).

## What Changes
- Create README.md with installation, usage, commands table, architecture overview
- Update docs/module-template.md to reflect current patterns (demo namespace, 48 examples)
- Create Claude Code skill `.claude/skills/add-module/` for automated module scaffolding
- Skill includes templates, validation, and best practices from existing demos

## Impact
- **Affected specs**:
  - Creates new capability `project-documentation`
  - Creates new capability `developer-tooling`
- **Affected code**:
  - New: `README.md` (project root)
  - Modified: `docs/module-template.md` (updated patterns)
  - New: `.claude/skills/add-module/SKILL.md` (automation skill)
- **Dependencies**: None (documentation only)
- **Testing**: README renders correctly, skill is discoverable by Claude
