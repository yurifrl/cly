# Sketch Manifest

## Design Direction
Dark, terminal-adjacent, glanceable. The sidebar shows **one card — the focused
surface's omp session as a summary** — never a list. Change surfaces and the card
swaps. Card carries full context for a cold reader: Session Goal, Last Request,
Response, Needs Action (orange only when the user is actually needed), plus a
caveman-tone timeline block underneath. No lorem ipsum; mock data mirrors real
omp sessions.

## Reference Points
- Current omp-cards sidebar (pipe-joined tagged segments) — being replaced
- cmux focused-surface signal (the card mirrors whatever surface is in focus)

## Sketches

| # | Name | Design Question | Winner | Tags |
|---|------|-----------------|--------|------|
| 001 | omp-sidebar-card | How should the focused session's summary card + timeline read at a glance? | null | [layout, card, sidebar, summary] |
