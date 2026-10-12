# Skill Creation Detailed Workflow

This document contains the complete step-by-step process for creating effective skills. For a quick overview, see SKILL.md.

## Skill Creation Process

To create a skill, follow the "Skill Creation Process" in order, skipping steps only if there is a clear reason why they are not applicable.

### Step 1: Understanding the Skill with Concrete Examples

Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.

To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.

For example, when building an image-editor skill, relevant questions include:

- "What functionality should the image-editor skill support? Editing, rotating, anything else?"
- "Can you give some examples of how this skill would be used?"
- "I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?"
- "What would a user say that should trigger this skill?"

To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.

Conclude this step when there is a clear sense of the functionality the skill should support.

### Step 2: Planning the Reusable Skill Contents

To turn concrete examples into an effective skill, analyze each example by:

1. Considering how to execute on the example from scratch
2. Determining the appropriate level of freedom for Claude
3. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly

**Match specificity to task risk:**

- **High freedom (text instructions)**: Multiple valid approaches exist; context determines best path (e.g., code reviews, troubleshooting, content analysis)
- **Medium freedom (pseudocode with parameters)**: Preferred patterns exist with acceptable variation (e.g., API integration patterns, data processing workflows)
- **Low freedom (exact scripts)**: Operations are fragile, consistency critical, sequence matters (e.g., PDF rotation, database migrations, form validation)

Example: When building a `pdf-editor` skill to handle queries like "Help me rotate this PDF," the analysis shows:

1. Rotating a PDF requires re-writing the same code each time
2. A `scripts/rotate_pdf.py` script would be helpful to store in the skill

Example: When designing a `frontend-webapp-builder` skill for queries like "Build me a todo app" or "Build me a dashboard to track my steps," the analysis shows:

1. Writing a frontend webapp requires the same boilerplate HTML/React each time
2. An `assets/hello-world/` template containing the boilerplate HTML/React project files would be helpful to store in the skill

Example: When building a `big-query` skill to handle queries like "How many users have logged in today?" the analysis shows:

1. Querying BigQuery requires re-discovering the table schemas and relationships each time
2. A `references/schema.md` file documenting the table schemas would be helpful to store in the skill

To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.

### Step 3: Initializing the Skill

At this point, it is time to actually create the skill.

Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.

**For global skills** (in `~/DotFiles/home/.claude/skills/`), create the directory structure manually:

```bash
# Create skill directory
mkdir -p ~/DotFiles/home/.claude/skills/skill-name

# Create SKILL.md with frontmatter template
cat > ~/DotFiles/home/.claude/skills/skill-name/SKILL.md << 'EOF'
---
name: skill-name
description: TODO - Clear explanation of function and use cases (max 200 chars)
category: TODO
tags: [TODO]
---

# Skill Name

TODO - Add skill content here

## About

TODO - Explain what this skill does

## Usage

TODO - Explain how to use this skill

## Reference Files

For detailed workflow, see `references/detailed_workflow.md`
EOF

# Optionally create resource directories
mkdir -p ~/DotFiles/home/.claude/skills/skill-name/{scripts,references,assets}
```

**For marketplace skills**, use the `init_skill.py` script from the claude-code-skills repository:

```bash
scripts/init_skill.py <skill-name> --path <output-directory>
```

The script:

- Creates the skill directory at the specified path
- Generates a SKILL.md template with proper frontmatter and TODO placeholders
- Creates example resource directories: `scripts/`, `references/`, and `assets/`
- Adds example files in each directory that can be customized or deleted

After initialization, customize or remove the generated SKILL.md and example files as needed.

### Step 4: Edit the Skill

When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Claude to use. Focus on including information that would be beneficial and non-obvious to Claude. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Claude instance execute these tasks more effectively.

#### Start with Reusable Skill Contents

To begin implementation, start with the reusable resources identified above: `scripts/`, `references/`, and `assets/` files. Note that this step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates to store in `assets/`, or documentation to store in `references/`.

Also, delete any example files and directories not needed for the skill. The initialization script creates example files in `scripts/`, `references/`, and `assets/` to demonstrate structure, but most skills won't need all of them.

**When updating an existing skill**: Scan all existing reference files to check if they need corresponding updates. New features often require updates to architecture, workflow, or other existing documentation to maintain consistency.

#### Reference File Naming

Filenames must be self-explanatory without reading contents.

**Pattern**: `<content-type>_<specificity>.md`

**Examples**:

- ❌ `commands.md`, `cli_usage.md`, `reference.md`
- ✅ `script_parameters.md`, `api_endpoints.md`, `database_schema.md`

**Test**: Can someone understand the file's contents from the name alone?

#### Update SKILL.md

**Writing Style:** Write the entire skill using **imperative/infinitive form** (verb-first instructions), not second person. Use objective, instructional language (e.g. "To accomplish X, do Y" rather than "You should do X" or "If you need to do X"). This maintains consistency and clarity for AI consumption.

To complete SKILL.md, answer the following questions:

1. What is the purpose of the skill, in a few sentences?
2. When should the skill be used?
3. In practice, how should Claude use the skill? All reusable skill contents developed above should be referenced so that Claude knows how to use them.

### Step 5: Security Review

Before packaging or distributing a skill, run the security scanner to detect hardcoded secrets and personal information:

```bash
# Required before packaging
python scripts/security_scan.py <path/to/skill-folder>

# Verbose mode includes additional checks for paths, emails, and code patterns
python scripts/security_scan.py <path/to/skill-folder> --verbose
```

**Detection coverage:**

- Hardcoded secrets (API keys, passwords, tokens) via gitleaks
- Personal information (usernames, emails, company names) in verbose mode
- Unsafe code patterns (command injection risks) in verbose mode

**First-time setup:** Install gitleaks if not present:

```bash
# macOS
brew install gitleaks

# Linux/Windows - see script output for installation instructions
```

**Exit codes:**

- `0` - Clean (safe to package)
- `1` - High severity issues
- `2` - Critical issues (MUST fix before distribution)
- `3` - gitleaks not installed
- `4` - Scan error

**Remediation for detected secrets:**

1. Remove hardcoded secrets from all files
2. Use environment variables: `os.environ.get("API_KEY")`
3. Rotate credentials if previously committed to git
4. Re-run scan to verify fixes before packaging

**Note**: For global skills in `~/DotFiles/home/.claude/skills/`, security scanning is less critical unless sharing publicly, but still recommended as best practice.

### Step 6: Packaging a Skill

**For global skills**: Packaging is not required since they're loaded directly from `~/DotFiles/home/.claude/skills/`. Skip to Step 7.

**For marketplace skills**: Once the skill is ready, it should be packaged into a distributable zip file:

```bash
scripts/package_skill.py <path/to/skill-folder>
```

Optional output directory specification:

```bash
scripts/package_skill.py <path/to/skill-folder> ./dist
```

The packaging script will:

1. **Validate** the skill automatically, checking:
   - YAML frontmatter format and required fields
   - Skill naming conventions and directory structure
   - Description completeness and quality
   - **Path reference integrity** - all `scripts/`, `references/`, and `assets/` paths mentioned in SKILL.md must exist

2. **Package** the skill if validation passes, creating a zip file named after the skill (e.g., `my-skill.zip`) that includes all files and maintains the proper directory structure for distribution.

**Common validation failure:** If SKILL.md references `scripts/my_script.py` but the file doesn't exist, validation will fail with "Missing referenced files: scripts/my_script.py". Ensure all bundled resources exist before packaging.

If validation fails, the script will report the errors and exit without creating a package. Fix any validation errors and run the packaging command again.

### Step 7: Iterate

After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.

**Iteration workflow:**

1. Use the skill on real tasks
2. Notice struggles or inefficiencies
3. Identify how SKILL.md or bundled resources should be updated
4. Implement changes and test again

**Refinement filter:** Only add what solves observed problems. If best practices already cover it, don't duplicate.
