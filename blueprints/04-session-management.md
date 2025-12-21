# Session Management

Named CLI sessions with Zellij tab integration.

---

## What It Does

**Named sessions** - Give CLI sessions memorable names
**Zellij sync** - Tab name updates to match session
**Environment export** - Sets `CLAUDE_SESSION_NAME` for downstream tools

---

## Usage

```bash
# Explicit name
cly --name WorkProject

# Auto-generated name
cly --name

# From environment
CLAUDE_SESSION_NAME=WorkProject cly
```

**What happens:**
- Prints: `🏷️  Session: WorkProject`
- Sets Zellij tab name to `WorkProject`
- Exports: `CLAUDE_SESSION_NAME=WorkProject`

---

## Session Naming

**Sources (priority order):**
- `--name NAME` flag
- `CLAUDE_SESSION_NAME` env var
- `--name` (no value) auto-generates

**Auto-generated format:**
- Two random words in TitleCase
- Examples: `QuickFox`, `BrightOwl`, `SwiftBear`
- Word pools: adjectives + animals

---

## Zellij Integration

**Tab naming:**
- Detects `$ZELLIJ` environment
- Uses `zellij action rename-tab`
- Silent fallback if not in Zellij

**Escape sequence (alternative):**
```
\x1b]0;SessionName\x07
```

---

## Environment

**Exports:**
- `CLAUDE_SESSION_NAME` - Current session name

**Reads:**
- `CLAUDE_SESSION_NAME` - Default name if no flag


---

References:

- /Users/yuri/DotFiles/home/.config/fish/functions/enforce_kube_context.fish
    - Use when in doubt about implementation defatils
