---
sketch: 001
name: omp-sidebar-card
question: "How should the focused surface's session summary card + timeline read at a glance?"
winner: null
tags: [layout, card, sidebar, summary]
---

# Sketch 001: omp sidebar card

## Design Question
One card = the focused surface's omp session. No list. The card's primary job:
answer **what did the user last request on this session** — LAST REQUEST is the
hero (first and biggest after the header). Full summary model confirmed with
Yuri:

```
LAST REQUEST      ← hero: what the user last asked, 13.5px bright
---
Goal              ← why the session exists, dim
---
Response          ← what the ai last did about it, dim
---
Needs Action      ← what the user needs to answer (orange only when actually needed)
```

Underneath, a separate **Timeline** block: one-line checkpoints per turn, caveman
tone, 👤 for user turns and 🤖 for ai turns.

## What the mock shows
- Fake cmux surface strip (vpn / WIP / meca-pi / ai tests / Incident Admin / Group 3)
- Card + timeline follow focus: click a surface, both blocks swap
- **Needs Action** goes orange only when the session actually waits on the user —
  the single glance signal
- Group 3 has no session → dashed empty states for card and timeline
- Width selector 320 / 370 / 420 (320 = real sidebar width)

## What to Look For
- Does LAST REQUEST read as the hero — the eye lands on it before anything else?
- Is the supporting stack (Goal / Response / Needs Action) clearly secondary at
  320px without feeling like a form?
- Does the timeline read as scannable history, not prose?
- Does Needs Action stand out exactly when it should?
