---
name: skill-creation
description: Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.
category: Development
tags: [skills, meta, development]
---

# Skill Creator

This skill provides guidance for creating effective skills.

## About Skills

Skills are modular, self-contained packages that extend Claude's capabilities by providing specialized knowledge, workflows, and tools. Think of them as "onboarding guides" for specific domains or tasks—they transform Claude from a general-purpose agent into a specialized agent equipped with procedural knowledge that no model can fully possess.

### What Skills Provide

1. Specialized workflows - Multi-step procedures for specific domains
2. Tool integrations - Instructions for working with specific file formats or APIs
3. Domain expertise - Company-specific knowledge, schemas, business logic
4. Bundled resources - Scripts, references, and assets for complex and repetitive tasks

### Anatomy of a Skill

Every skill consists of a required SKILL.md file and optional bundled resources:

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter metadata (required)
│   │   ├── name: (required)
│   │   └── description: (required)
│   └── Markdown instructions (required)
└── Bundled Resources (optional)
    ├── scripts/          - Executable code (Python/Bash/etc.)
    ├── references/       - Documentation intended to be loaded into context as needed
    └── assets/           - Files used in output (templates, icons, fonts, etc.)
```

#### SKILL.md (required)

**Metadata Quality:** The `name` and `description` in YAML frontmatter determine when Claude will use the skill. Be specific about what the skill does and when to use it. Use the third-person (e.g. "This skill should be used when..." instead of "Use this skill when...").

#### Bundled Resources (optional)

##### Scripts (`scripts/`)

Executable code (Python/Bash/etc.) for tasks that require deterministic reliability or are repeatedly rewritten.

- **When to include**: When the same code is being rewritten repeatedly or deterministic reliability is needed
- **Example**: `scripts/rotate_pdf.py` for PDF rotation tasks
- **Benefits**: Token efficient, deterministic, may be executed without loading into context
- **Note**: Scripts may still need to be read by Claude for patching or environment-specific adjustments

##### References (`references/`)

Documentation and reference material intended to be loaded as needed into context to inform Claude's process and thinking.

- **When to include**: For documentation that Claude should reference while working
- **Examples**: `references/finance.md` for financial schemas, `references/mnda.md` for company NDA template, `references/policies.md` for company policies, `references/api_docs.md` for API specifications
- **Use cases**: Database schemas, API documentation, domain knowledge, company policies, detailed workflow guides
- **Benefits**: Keeps SKILL.md lean, loaded only when Claude determines it's needed
- **Best practice**: If files are large (>10k words), include grep search patterns in SKILL.md
- **Avoid duplication**: Information should live in either SKILL.md or references files, not both. Prefer references files for detailed information unless it's truly core to the skill—this keeps SKILL.md lean while making information discoverable without hogging the context window. Keep only essential procedural instructions and workflow guidance in SKILL.md; move detailed reference material, schemas, and examples to references files.

##### Assets (`assets/`)

Files not intended to be loaded into context, but rather used within the output Claude produces.

- **When to include**: When the skill needs files that will be used in the final output
- **Examples**: `assets/logo.png` for brand assets, `assets/slides.pptx` for PowerPoint templates, `assets/frontend-template/` for HTML/React boilerplate, `assets/font.ttf` for typography
- **Use cases**: Templates, images, icons, boilerplate code, fonts, sample documents that get copied or modified
- **Benefits**: Separates output resources from documentation, enables Claude to use files without loading them into context

##### Privacy and Path References

**CRITICAL**: Skills intended for public distribution must not contain user-specific or company-specific information:

- **Forbidden**: Absolute paths to user directories (`/home/username/`, `/Users/username/`, `/mnt/c/Users/username/`)
- **Forbidden**: Personal usernames, company names, department names, product names
- **Forbidden**: OneDrive paths, cloud storage paths, or any environment-specific absolute paths
- **Forbidden**: Hardcoded skill installation paths like `~/.claude/skills/` or `/Users/username/Workspace/claude-code-skills/`
- **Allowed**: Relative paths within the skill bundle (`scripts/example.py`, `references/guide.md`)
- **Allowed**: Standard placeholders (`~/workspace/project`, `username`, `your-company`)
- **Best practice**: Reference bundled scripts using simple relative paths like `scripts/script_name.py` - Claude will resolve the actual location

##### Versioning

**CRITICAL**: Skills should NOT contain version history or version numbers in SKILL.md:

- **Forbidden**: Version sections (`## Version`, `## Changelog`, `## Release History`) in SKILL.md
- **Forbidden**: Version numbers in SKILL.md body content
- **Rationale**: SKILL.md should be timeless content focused on functionality
- **For global skills**: Track versions in git history or external changelog if needed

### Progressive Disclosure Design Principle

Skills use a three-level loading system to manage context efficiently:

1. **Metadata (name + description)** - Always in context (~100 words)
2. **SKILL.md body** - When skill triggers (<5k words)
3. **Bundled resources** - As needed by Claude (Unlimited*)

*Unlimited because scripts can be executed without reading into context window.

### Skill Creation Best Practice

Anthropic has wrote skill authoring best practices, retrieve it before creating or updating skills: https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices.md

## ⚠️ CRITICAL: Edit Skills at Source Location

**NEVER edit skills in `~/.claude/plugins/cache/`** — that's a read-only cache directory. All changes there are:

- Lost when cache refreshes
- Not synced to source control
- Wasted effort requiring manual re-merge

**ALWAYS verify you're editing the source repository:**

```bash
# WRONG - cache location (read-only copy)
~/.claude/plugins/cache/daymade-skills/my-skill/1.0.0/my-skill/SKILL.md

# RIGHT - source repository (global skills)
~/DotFiles/home/.claude/skills/my-skill/SKILL.md

# RIGHT - source repository (local project skills)
/path/to/project/.claude/skills/my-skill/SKILL.md
```

**Before any edit**, confirm the file path does NOT contain `/cache/` or `/plugins/cache/`.

## Quick Start

For the complete step-by-step process, see `references/detailed_workflow.md`.

**Brief overview:**

1. **Understand** - Gather concrete examples of how the skill will be used
2. **Plan** - Identify what scripts, references, and assets are needed
3. **Initialize** - Create skill directory structure (manual for global skills, `init_skill.py` for marketplace)
4. **Edit** - Create reusable resources and write SKILL.md using imperative form
5. **Security review** - Run `security_scan.py` to detect secrets and personal info (marketplace skills)
6. **Package** - Run `package_skill.py` to validate and zip (marketplace skills only)
7. **Iterate** - Test and refine based on real usage

## Reference Files

- `references/detailed_workflow.md` - Complete step-by-step skill creation process with examples

## Related Resources

- [Anthropic Engineering: Equipping Agents](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)
- [Claude Support: Creating Skills](https://support.claude.com/en/articles/12512198-how-to-create-custom-skills)
- [Anthropic Docs: Best Practices](https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices.md)
