# Port claudext Session Naming

**Goal**: Port the `--name` session naming feature from Fish claudext

**Source**: `~/DotFiles/home/.config/fish/functions/claudext.fish` (case '*' section)

---

## What claudext Does

When you run `claudext --name [NAME]` or just `claudext`:

1. **Parse `--name` flag** from arguments
2. **Get session name**:
   - If `--name VALUE` provided → use VALUE
   - If `--name` without value → generate random ColorAnimal
   - If no `--name` at all → generate random ColorAnimal
3. **Set environment variable**: `CLAUDE_SESSION_NAME={name}`
4. **Display** session name to user
5. **Rename Zellij tab** (optional, if in Zellij)
6. **Pass remaining args** to `claude` CLI

---

## Key Behavior from Fish Code

```fish
# Default case - handles --name flag
case '*'
    # Parse --name flag
    if --name provided with value:
        session_name = value
    else:
        session_name = __generate_session_name()  # Random ColorAnimal

    # Set env var
    set -gx CLAUDE_SESSION_NAME "$session_name"

    # Show name
    echo "🏷️  Session: $session_name"

    # Rename tab (if in Zellij)
    zellij action rename-tab "$session_name"

    # Run Claude with remaining args
    claude $claude_args
```

---

## Random Name Generator

**Function**: `__generate_session_name()`

**Logic**:
- Pick random color from 14 options
- Pick random animal from 16 animals
- Capitalize each word
- Concatenate (no separator)

**Word lists**:
- Colors: red, blue, green, yellow, purple, orange, pink, cyan, magenta, lime, teal, navy, maroon, olive (14)
- Animals: cat, dog, fox, wolf, bear, lion, tiger, shark, eagle, hawk, dove, owl, rabbit, deer, mouse, rat (16)

**Output**: RedWolf, BlueFox, GreenTiger, YellowEagle, etc. (224 combinations)

---

## What to Port to CLY

A utility command that:
1. Generates random ColorAnimal names
2. Optionally accepts user-provided name
3. Sets `CLAUDE_SESSION_NAME` environment variable
4. Can be used standalone or with Claude CLI

**Command**:
```bash
cly name [NAME]
```

**Behavior**:
- `cly name` → Generate random ColorAnimal, set env var, output name
- `cly name MySession` → Use "MySession", set env var, output name
- Output format: Just the name (for scripting)

**Usage with Claude**:
```bash
# Generate random name and start Claude
cly name && claude

# Use specific name
cly name MyProject && claude

# In Fish
set -gx CLAUDE_SESSION_NAME (cly name)
claude
```

---

## Requirements

### Name Generation
- If argument provided: use it as-is
- If no argument: generate random ColorAnimal
- ColorAnimal format: Capitalize(color) + Capitalize(animal)
- Random selection from word lists

### Environment Variable
- Always set `CLAUDE_SESSION_NAME` to generated/provided name
- Export so child processes (like Claude CLI) can see it

### Output
- Print the name to stdout (for scripting/capture)
- Minimal output (just the name, no extra text)

### Word Lists
- 14 colors (exact list from claudext)
- 16 animals (exact list from claudext)
- No adjectives (claudext has them but doesn't use them)

---

## Out of Scope

**Not porting**:
- Workspace/folder management
- Session persistence to JSON file
- Session resume functionality
- Calling Claude CLI directly
- Zellij integration
- File creation
- Archive functionality

**Just**: Name generation + env var setting

---

## Module Location

`modules/name/` (utility module, not demo)
