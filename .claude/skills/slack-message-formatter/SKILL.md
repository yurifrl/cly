---
name: slack-message-formatter
description: This skill should be used when the user asks to parse, format, organize, or work with a message. It asks if the message is for Slack and formats using Slack's minimal markdown with natural conversational flow.
---

# Slack Message Formatter

When the user asks you to parse, format, organize, or do anything with a message, **always ask if this is for Slack** before proceeding.

If yes, format the message following these rules:

## Slack Formatting Rules

### Text Formatting
- **Bold**: Single asterisk `*bold*` (NOT double `**`)
- **Italic**: Underscore `_italic_`
- **Strikethrough**: Tilde `~strike~`
- **Code**: Backticks `` `code` ``
- **Code block**: Triple backticks
- **Quote**: Prefix line with `>`

### Structure
- **Minimal line breaks**: Don't break lines excessively. Keep paragraphs flowing naturally like human conversation
- **No markdown separators**: Never use `---` or `===`
- **Natural flow**: Write like you're typing in a chat, not composing formal documentation

### Lists
- No special list syntax - use plain text with line breaks
- Use `-` or `•` for bullets if needed
- **Sublists**: Indent with 4 spaces (one tab)

### Mentions & Links
- **User mentions**: `@username` (Slack converts to `<@USER_ID>`)
- **Links**: `<URL|display text>` or just raw URL
- **Email**: `<mailto:email@domain.com|Name>`

### Line Breaks
- Use actual line breaks (not `\n` strings)
- Keep related content together
- Separate distinct topics with single blank line

## Formatting Style

**Key principle**: Write like a human in a chat app, not like you're writing documentation.

**Good example** (what Slack messages look like):
```
Hey Greg,

Happy New Year! Apologies if I missed your email—not the most tech-savvy with email management.

I'd like to discuss a few topics:

*Custom DNS* - We talked about this as a WIP feature. I'd like to know more about the current status.

*Cloudflare Access replacement* - Looking for an alternative to give external users access to specific internal services. We're using it now but feel we're underutilizing it. A showcase would be helpful.

*People to loop in:*
    @supergrilo - Behind our IAM team
    @Diego Fagundes - Head of SRE, would appreciate the context
```

**Bad example** (too formal, over-structured):
```
---

Hey Greg,

Happy New Year!

Apologies if I overlooked your message—not the most tech-savvy with notifications.

I'd like to discuss a few topics:

**Custom DNS**
- We talked about this as a WIP feature
- I'd like to know more about the current status

**Cloudflare Access replacement**
- Looking for an alternative
- Give external users access to specific internal services

---

People to loop in:
- @supergrilo - Behind our IAM team
- @Diego Fagundes - Head of SRE

---
```

## Workflow

1. When user asks to work with a message, ask: "Is this for Slack?"
2. If yes, format using the rules above
3. Use `pbcopy` to pipe the result to clipboard
4. Keep it conversational and natural
