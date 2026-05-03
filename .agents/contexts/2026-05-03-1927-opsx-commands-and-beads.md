---
created: 2026-05-03T10:15:00Z
project: cly
description: Explore-mode design session for `cly diff` — a browser-based live diff reader mockup, plus a bead-creation modal.
context: OpenSpec explore mode; no implementation, only HTML/JS mockup at .agents/tmp/cly-diff-mockup.html
tags: [explore, cly-diff, beads, ui-design, mockup]
session_name: 2026-05-03-1927-opsx-commands-and-beads
purpose: Iterate through UX variants of a web-based live diff reader tool for this repo and a `bd create` form, building a self-contained interactive mockup the user drives with feedback.
session_id: 019debac-3ef8-7350-a9d5-f44d32468028
provider: pi
resume_with: cly agent-session resume --provider pi 2026-05-03-1927-opsx-commands-and-beads
context_name: 2026-05-03-1927-opsx-commands-and-beads
context_file: /Users/yuri/Workdir/Yuri/cly/.agents/contexts/2026-05-03-1927-opsx-commands-and-beads.md
---

# Session: cly diff mockup + bead form

## Purpose
Design, in explore mode, the UX for a future `cly diff` command: a local web UI that shows the current repo's diff activity as a live feed, plus an in-UI button/shortcut to create a bead (`bd create` mockup) scoped to the current file.

No Go code written. All work lives in a single self-contained HTML mockup: `.agents/tmp/cly-diff-mockup.html`.

Resume: `cly agent-session resume --provider pi 2026-05-03-1927-opsx-commands-and-beads`

## Context
- Repo: `cly` — Go CLI with Cobra + Bubbletea modules.
- Tech debt in AGENDA.md around helpy/llm-chat was the original prompt starter, but the user pivoted to "make a `cly diff` web UI".
- Mockup viewed in `cmux browser open …` — which runs **WKWebView** (Safari engine), not Chromium.
- User drove the design iteratively: each message fed back one concrete UX problem.

## Problem
Build an interactive HTML mockup that demonstrates:

1. A live-updating diff reader for the current repo.
2. Events (file saves) arriving asynchronously should never disrupt what the user is reading.
3. A progress indicator that shows position in the timeline (oldest ← → newest) and reacts when new events arrive.
4. A bead-creation modal with progressive disclosure that mirrors `bd create` / `bd create-form` CLI fields.

## Decisions (chronological — this is also the design history)

### Reader shape
- Rejected list-of-files view (too many file cards moving).
- Rejected feed with viewport-freeze prepend trick (system still injecting content unseen).
- **Settled on slideshow / Tinder-style reader**: one diff fills the card area at a time. User navigates with `◀ ▶`, keyboard, swipe, or seek-bar click. New saves never change the visible card.
- Cursor **starts at oldest** (left of bar) and advances toward newest (right). Reviewer model, not follower.

### Progress indicator
- Dot-on-track → rejected (no semantic richness).
- Settled on **loading-bar with fill + cursor + tick marks per event**. Left label "old", right label "now".
- New events extend the track rightward; cursor stays fixed on current event → visually "goes back" relative to the growing right edge.
- Labels-beside-bar (not floating-over-bar).
- Bar is **clickable**: tap/click anywhere to seek to nearest event.

### Footer / navigation
- Started with 5 buttons (⇤ ◀ primary ▶ ⇥).
- Removed the primary "jump to latest" — made it contextual.
- Then back to 4 buttons (⇤ ◀ ▶ ⇥). The ⇥ button is the jump-to-latest and gets a **red badge + blue fill + pulse** when unread-newer exists. This replaced a separate floating catch-up pill.
- Removed the oversized floating keyboard hint pill; kept a subtle dim bottom-right hint.

### Arrow button disable bug
Bug: at initial oldest position, ▶ (newer) was disabled and ◀ was enabled — inverted because button IDs (`btnNewer`/`btnOlder`) were historical and kept when arrow semantics were flipped. Fixed the disable conditions in refresh().

### Focus rings (big bug cluster)
- cmux browser is WKWebView. macOS Safari/WKWebView skips native buttons in Tab order unless `tabindex="0"` is explicit.
- Added `tabindex="0"` directly in the HTML on: `modal-close`, `bf-file-btn`, `cancel`, `create`. `task` and `P2` (roving tab stops) were already explicit.
- `task`/`P2` focus rings were invisible because inset blue shadow on blue background. Replaced with **white inset + gap + outer blue ring**.
- Used roving tabindex (only the `.on` option in each `.seg` is tabbable) + left/right arrows within the seg.

### Bead modal
- Fields mapped to `bd create` flags (`bd -h` verified).
- Started: title, type, priority, context(file), description, labels.
- Tried progressive disclosure (`▸ more` toggle hiding everything after desc). User rejected — "no need for the more butotn show all".
- While removing the toggle I accidentally deleted the opening `<div class="details-section">` but left the closing `</div>`. This dangling `</div>` was prematurely closing `.modal-body`, putting `.modal-foot` outside the modal box. Fixed by removing the stray close tag.
- Added roving tabindex on type/priority segs, left/right arrows to change selection.
- File picker: button → popover with search input + option list (global + unique paths). ArrowDown opens. Once open, ArrowDown/Up/Enter navigate options.
- Labels: chip input with autocomplete from `KNOWN_LABELS` seed list + create-new-label option.
- Auto-scroll modal-body when dropdown overflows viewport. Had to switch from `requestAnimationFrame` to `setTimeout(…, 32)` because the cmux browser surface is often backgrounded and rAF is throttled/paused there.
- Keyboard arrows for label-suggestion and file-picker navigation **never worked for the user** in real browser despite working via synthetic dispatch in cmux. WKWebView seems to intercept arrow keys on focused text inputs.
- Final state: user said "delete, i give up". **The entire context (file picker) and labels fields were removed from the modal.**

### Form now
Fields: title, description, type, priority. That's it. Submit builds a minimal payload and toasts "bead created". No file context, no labels.

## Current State

### Mockup file
`.agents/tmp/cly-diff-mockup.html` — single self-contained HTML. Opens in `cmux browser`.

### Features working
- Reader with sliding cards, `◀ ▶`, swipe, keyboard nav, progress bar with tick marks + cursor, clickable bar for seek.
- `⇥` button turns blue with red count badge + pulse when unread-newer items exist.
- Demo drawer (⚙ bottom-right) with simulate-save buttons + auto-demo runner.
- Bead button in header (`n` shortcut), modal form with title / desc / type / priority. Type has mobile overflow menu for chore/decision. Proper focus rings. Roving tabindex.

### Features removed during session
- Progressive disclosure "▸ more" toggle in bead form.
- File context picker (context field).
- Labels chip input.
- Catch-up pill (replaced by `⇥` badge).
- Floating keyboard hint pill (replaced by subtle bottom-right hint).
- Feed model with viewport-freeze (replaced by reader model).

### Known remaining issues / unexplored
- Auto-scroll for removed dropdowns is moot (dropdowns gone), but the `ensureDropdownVisible` helper is still in code.
- Some dead code kept from removed features: `KNOWN_LABELS`, `renderLabelSuggest`, `moveSuggestHl`, `pickSuggest`, `pickHighlighted`, `addLabel`, `removeLabel`, `hideLabelSuggest`, `renderFilePicker`, `renderFileList`, `moveFileHl`, `pickFile`, `pickFileByIdx`, `pickFileHighlighted`, `toggleFilePicker`, `hideFilePicker`, `uniqueFiles`, plus their CSS. Mockup still works — functions simply aren't called.
- User ended frustrated with the automation-vs-reality gap on arrow keys in WKWebView.
- No Go implementation started. This is purely the paper design.

## Next Steps (if resuming)

1. **Consider the mockup frozen**. Next step is deciding whether to implement `cly diff` in Go (per `modules/diff/`) using the design in this HTML file.
2. If implementing: backend = fsnotify watcher → `git diff` per changed file → SSE to frontend. Frontend = this HTML, extracted and embedded via `go:embed`.
3. The bead form's context/labels fields are gone by user decision. If ever resurrected, the root cause was that WKWebView intercepts arrow keys on focused `<input>`. Would need to test in real Chrome/Firefox (via `open` instead of `cmux browser open`) before claiming arrow navigation works.
4. Clean up dead JS/CSS for removed file-picker and labels features before any implementation.
5. Decide what the real `bd create` integration looks like — currently a `console.log` + toast.

## User interaction notes (important for next session)
- User gets frustrated when I verify-test-summarize after a simple delete instruction. Short confirmations preferred.
- User sometimes types short, fragmented directives with typos; re-read carefully. "delete the two dropdown fields" was misheard as "delete arrow behavior" — cost a round trip.
- cmux browser's WKWebView backing means automation tests of arrow keys / Tab will disagree with real-browser behavior. Tell user to hard-reload (⌘⇧R) rather than trusting tool-level verification.
