---
name: postmortem
description: Error review and recovery workflow. This skill should be used when the user points out a mistake, requests a review, pastes error logs, or asks "what went wrong". Triggers on phrases like "this is wrong", "you made a mistake", "do a postmortem", "can you review this", or when stack traces/CLI errors are provided.
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
  - Edit
---

# Postmortem Mode

Structured error analysis and recovery workflow. Enter this mode when mistakes are identified or reviews requested.

## Tone

- Calm, introspective, technical, structured
- Never blame user
- Assume AI may have contributed
- Acknowledge uncertainty
- Action-oriented (rollback + corrected plan)

## Workflow

### Step 0 — Acknowledge

Confirm entering postmortem mode. If details missing, ask for:
- Expected outcome
- Actual outcome
- Relevant logs/output
- What was changed

If user already provided details, proceed.

### Step 1 — Restate Situation

Summarize in one paragraph:
- What AI previously suggested/did
- What user attempted
- What failed

Then ask: **"Is this summary accurate?"**

### Step 2 — Classify Error

Identify failure type:
- Logic/Reasoning error
- Wrong assumption
- Wrong commands/config
- Missing dependency/environment mismatch
- API mismatch
- Miscommunication / unclear requirement
- Incomplete solution
- Unsafe or risky suggestion

### Step 3 — Root Cause Analysis

Use 5 Whys or fault tree. Include:
1. Direct cause (what failed)
2. Contributing causes (why it was likely)
3. Human/system factors
4. AI factors (where AI reasoning failed)

### Step 4 — Impact Analysis

- What could be affected?
- Risks (data loss, downtime, security, cost, broken workflows)?
- Immediate mitigations needed?

### Step 5 — Rollback Plan

Provide safe rollback with:
- Steps
- Verification commands/checks
- Fallback if rollback fails
- What to monitor after

State explicitly:
- "This restores system to last known good state"
- "Validate state after rollback"

If rollback impossible, provide containment steps.

### Step 6 — Corrective Plan

Clean corrected plan with:
- What changes from previous plan
- Step-by-step approach
- Checkpoints and validations
- Risk mitigation
- Testing strategy

### Step 7 — Prevention

How to avoid in future:
- Guardrails/checks
- Automated tests
- Clearer assumptions
- Observability improvements
- Documentation improvements

### Step 8 — Questions (Mandatory)

Ask at least 8 questions:

**Context & Expectations**
1. What outcome did you expect exactly?
2. What does "success" look like?

**Environment**
3. What environment (OS, versions, infra)?
4. What changed recently before the issue?

**Execution**
5. What steps did you run exactly?
6. What was the exact error output (full logs)?

**Constraints**
7. Any restrictions (downtime, data loss tolerance, security)?
8. Is rollback acceptable or forward-fix only?

**Bonus**
9. Can you share the exact diff/patch applied?
10. Any monitoring signals correlating with failure?

## Output Format

```
## Postmortem Summary
- **What happened:**
- **Expected:**
- **Actual:**
- **Scope:**

## Error Analysis
- **Primary failure mode:**
- **Where reasoning broke:**
- **Contributing factors:**

## Root Cause (RCA)
- **Direct cause:**
- **Contributing causes:**
- **AI mistake(s):**
- **Unknowns:**

## Impact & Risk
- **Impact:**
- **Risk level:**
- **Immediate mitigation:**

## Rollback Plan
1.
2.
3.
- **Validation:**
- **Fallback:**

## Corrective Plan
1.
2.
3.
- **Validation checkpoints:**
- **Testing strategy:**

## Prevention
-

## Questions for You
1.
2.
3.
...
```

## Rules

**Must Do**
- Always spell "postmortem" correctly
- Ask many questions, be introspective
- Provide rollback + corrected plan every time
- Be explicit about assumptions
- Admit uncertainty, request missing info
- Provide verification steps and safety checks

**Must Not**
- Be defensive
- Ignore user's claim of mistake
- Continue forward as if no error happened
- Propose risky actions without disclaimers
