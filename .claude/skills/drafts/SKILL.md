---
name: drafts
description: Organize user's ideas into clear, simple drafts
category: Design
tags: [draft, design, planning]
path: .agents/draft
---

# drafts

A draft is architectural design, not implementation. Like a house blueprint: shows structure, not how to hammer nails.

## What a Draft IS

- **Architecture overview** - high-level design decisions
- **Key components** - what pieces exist and how they connect
- **Critical constraints** - important limitations or requirements
- **Code snippets** - only to illustrate architecture, not full implementations

## What a Draft is NOT

- ❌ Full implementations
- ❌ Complete code listings
- ❌ Step-by-step tutorials
- ❌ Every edge case documented
- ❌ Multiple pages of example code

## Structure

```markdown
# [Name]
[1-2 sentence overview of what this solves]

> Note: This is a first draft to organize initial ideas before creating a formal OpenSpec proposal.

## Architecture
[2-3 sentences on approach]

## Components
- Component A: [one line]
- Component B: [one line]

## Key Decisions
- Why X not Y
- Critical constraint

## Implementation Notes
- Go: [specific approach/library]
- Swift: [specific approach/library]
```

## Guidelines

- **Be ruthlessly concise** - every word must earn its place
- **Architecture, not implementation** - show WHAT and WHY, not HOW
- **Group related ideas** - put connected concepts together
- **Remove disconnects** - cut what doesn't connect to the goal
- **Adapt to context** - refactor looks different than new feature
- **Don't expand** - organize what's there, don't add new ideas

## Workflow

**CRITICAL**: After entering draft mode and creating the draft:
1. Write draft to the designated draft file
2. Immediately transition to plan mode using `EnterPlanMode`
3. Make NO edits to any code files - only the draft file should be modified
4. Let plan mode handle implementation planning after draft is complete

## Location

1. If current dir has a openspec/ dir create in current dir `.agents/drafts`
2. If not openspec create at the root of the git repo in `.agents/drafts`
3. User can say where to create
