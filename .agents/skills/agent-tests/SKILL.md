---
name: agent-tests
description: Agent-runnable e2e tests that validate workflows with real commands
category: Testing
tags: [test, e2e, agent, validation]
---

# Agent Tests

End-to-end tests that an agent (or human) can execute using real CLI calls. No mocking — run real commands, observe real outcomes, assert expected state.

## Location

```
.agents/tests/<test-name>/
├── TEST.md          # Required: test definition
├── test-*.sh        # Optional: executable test script
└── fixtures/        # Optional: test data
```

## TEST.md Format

```markdown
# Test: <Name>

## Context
[What this test validates and why]

## Workflow
1. [Step with real command]
2. [Step with real command]
3. ...

## Assertions

| Step | What to check | How |
|------|--------------|-----|
| ... | ... | `command to verify` |

## Commands
[Optional: bash script or inline commands that run the full test]

## Cleanup
[How to undo side effects, flags like --no-cleanup]
```

## Guidelines

- Tests use real tools (gh, kubectl, curl, etc.) — no mocks
- Each test is self-contained in its directory
- TEST.md is the source of truth, scripts are optional automation
- Include cleanup steps or --no-cleanup flag for inspection
- Use polling with timeouts for async workflows
- Print step-by-step progress when scripted
- Tests can accept arguments (target repo, environment, flags)
