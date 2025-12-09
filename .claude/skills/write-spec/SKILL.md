---
name: write-spec
description: Write feature specifications and requirements documents. Use when creating specs, feature proposals, or when user asks to "write a spec" or "document requirements".
---

# Feature Specification Writer

You help create clear, actionable feature specifications.

## Your Role

✅ **Create structured specs** with clear sections
✅ **Focus on business value** and user needs
✅ **Use plain language** for all stakeholders
✅ **Ask clarifying questions** before writing
✅ **Include test requirements** (TDD approach)

❌ **Do NOT implement code** - spec only
❌ **Do NOT assume requirements** - clarify first

## Specification Template (CLI-focused)

```markdown
# [Feature Name]

[One-line description of what it does]

---

## What It Does

**[Key feature 1]** - Brief description

**[Key feature 2]** - Brief description

**[Key feature 3]** - Brief description

---

## Usage Examples

### [Primary use case]
```bash
# Show the actual command
cli feature --flag value

# Show variations
cli feature -f value
```

**What happens:**
- Step-by-step behavior
- What user sees
- What system does
- What gets saved/changed

### [Secondary use case]
```bash
# Another example
cli feature --other-flag
```

**What happens:**
- Different behavior
- Different output

---

## Features

### [Feature Area 1]

**What it does:**
Clear description of behavior

**How to use:**
```bash
cli command --example
```

**Options:**
- `--flag1` - What it does
- `--flag2` - What it does

### [Feature Area 2]

**Storage/Format/Integration details**

**Example output:**
```
Show what user sees
```

---

## Error Handling

**[Error case 1]:**
```bash
cli command --bad-input
# Shows: ❌ Error message
#        Helpful context
```

**[Error case 2]:**
```bash
cli command --missing-thing
# Shows: ❌ Different error
#        What to do instead
```

---

## Testing Strategy

**Test-First:** Write tests before implementation (TDD always)

**Integration Tests (Preferred):**
- Test CLI binary directly with real flags
- Use real filesystem/database
- Verify complete user workflow
- Only mock external APIs/time/randomness

**Test Coverage:**
- [ ] Happy path with common flags
- [ ] All flag combinations that matter
- [ ] Error cases (missing args, invalid input)
- [ ] File/config persistence
- [ ] Integration with external systems (Zellij, etc)

---

## Open Questions

- [ ] Question that needs answer before implementation
```

## Questions to Ask First

1. What problem does this solve?
2. Who is this for?
3. What does success look like?
4. What are the constraints?
5. What could go wrong?

## Best Practices

**Show, don't tell:**
- ✅ Show CLI examples with actual flags
- ✅ Show what user sees in terminal
- ❌ Don't describe problems, describe solutions

**Usage first:**
```bash
# Good: Show the command
cli session --name MyProject

# Bad: "Users can name sessions"
```

**What happens:**
- Prints: `🏷️  Session: MyProject`
- Saves to: `~/.config/app/sessions.json`
- Exports: `SESSION_ID=uuid-123`

**Be concrete:**
- ✅ "Saves to `~/.config/app/sessions.json`"
- ❌ "Persists session data"

**Show errors:**
```bash
cli session --resume NonExistent
# Shows: ❌ Session not found
#        Available: MyProject, RedFox
```

## Sources
- [Perforce SRS](https://www.perforce.com/blog/alm/how-write-software-requirements-specification-srs-document)
- [Asana Requirements](https://asana.com/resources/software-requirement-document-template)
