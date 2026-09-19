# cly omp summary — Charm sidebar for live omp session summaries

Remake the `omp-focus-summary` bash POC as a native cly module: `cly omp summary`, a vertical TUI built with the Charm v2 stack, sized for the narrow cmux sidebar pane (~50–55 cols). Shows surface name + AI-generated summary of the omp session on the focused tab; updates async via a work queue; fully mouse-clickable.

## Context

**Proven in the POC** (`home/.local/bin/omp-focus-summary` in DotFiles, commits `c43b72c8`/`0008e18d`/`947a7fc0`):
- Focus discovery: `cmux --id-format both tree --all --json` → `.active.surface_id` (env/caller-independent; **`--id-format both` is mandatory** — plain output is refs-only, and `cmux sessions list --surface <ref>` silently matches nothing with a ref, UUID required).
- Session mapping: `cmux sessions list --surface <uuid> --all --json` → omp hook records (`~/.cmuxterm/omp-hook-sessions.json`). Reused tabs accumulate 600+ stale records → rank by `active_for_surface`, `runtime_status`, `updated_at_unix`; pick first.
- Session file: `~/.omp/agent/sessions/<munged-cwd>/<ts>_<sessionId>.jsonl`. Unique-id glob `*/*_<sid>.jsonl` beats cwd-munging (relative cwds). Entries: `type: message|custom|title|compaction|session_init…`; message roles `user|assistant|toolResult`; content blocks `text|toolCall{name,intent}`; assistant `stopReason: toolUse` marks tool turns; `usage.totalTokens` = context size; timestamps epoch-ms.
- No-arg invocation must mean "discover the app-focused tab" — no caller fallback (that was a POC bug: it silently summarized the wrong tab).

**cly already has**:
- `pkg/cmux/cmux.go` — Go wrapper around the cmux CLI (`BinaryAvailable()`, `Notify()`, `SetStatus()`). Extend, don't fork.
- `pkg/ai` (`ai.NewClient()` / `ai.NewClientWith(override)`) → `llm.Client` from `pkg/llm` — Anthropic/OpenAI/OpenRouter/Bedrock with the location-aware failover chain. Used by `modules/oi`, `modules/llm-chat`, `modules/git-commits`.
- Charm v2 conventions (`.agents/skills/charm-stack`): `charm.land/bubbletea/v2`, `View() tea.View`, `tea.KeyPressMsg`, declarative `v.MouseMode = tea.MouseModeAllMotion`, `tea.SetClipboard` for copy.
- Module pattern (`.agents/skills/add-module`): self-contained `modules/<name>/` + `Register(parent)`, never import other modules; `pkg/` only via `pkg/cmux`, `pkg/ai`, `pkg/style`.

**User requirements (verbatim intent)**:
1. Fits the cmux sidebar — vertical design (~53 cols wide, full height).
2. Shows: surface name; "last command" section (goal — what the user asked); "summary" section (last AI message summary — what the AI did/asks/expects).
3. Summary updates **async with AI**: on load, show last summary from `.omp/summary.json` immediately, then update. Switch to 3 tabs → 3 updates enqueue (dedup by tab); run updates in parallel (bounded workers).
4. Smart change detection: don't regenerate when nothing changed (fingerprint); state tracks what was already generated.
5. Charm stack, clickable, nice interface.
6. Display which AI model is used.

## UX Spec (sidebar layout, ~53 cols)

```
╭─ omp ─────────────────────────────────────╮
│ Mecatl Remote Subagent Extension Plan     │  ← surface/tab title (focused tab)
│ ● running · aihub/glm-5.3-flash · ctx 81k │  ← status dot, model badge (clickable: opens model menu later; copy on click)
├─ last command ────────────────────────────┤
│ "go" → fold Q1 into plan, push 4b5ff2a    │  ← user's last ask, condensed goal
├─ summary ─────────────────────────────────┤
│ Phase 1 complete and pushed. Verifying    │  ← what the AI is doing now
│ e2e; expects confirmation on .live…       │
├─ ai expects ──────────────────────────────┤
│ your decision on prod domain (.dev ok,    │  ← what the AI is asking/waiting for
│ .live needs CF Access)                    │
├─ tabs ────────────────────────────────────┤  ← all omp tabs, CLICK a row to focus it
│ ▶ 131 Mecatl Plan        fresh   2m ago   │     in cmux (surface.focus) — watcher
│   90  Slack reports      stale  34m ago   │     follows automatically
│   227 cly POC            …AI…             │
╰───────────────────────────────────────────╯
 ⟳ queue 2 · workers 2 · updated 2m ago     ← footer, live queue state
```

**Interactions** (mouse + keyboard equivalent for every action):
- **Click a tab row** → `cmux` focus that tab (`pkg/cmux.FocusSurface`); focused row gets `▶`; the watcher picks it up next tick. This is the "switch to 3 tabs" flow made explicit — click or alt-tab, both enqueue.
- **Click the "ai expects" block** (or `c` key) → copy last AI message to clipboard (`tea.SetClipboard`).
- **Click a section header** → collapse/expand it.
- Keys: `q`/`ctrl+c` quit, `j/k` move, `enter` focus selected tab, `c` copy, `r` force refresh (re-enqueue focused tab even if fresh), `p` pause/resume updates.
- Loading state per section: spinner line while a summary for the visible tab is generating; "stale" badge when fingerprint changed but AI hasn't caught up; relative timestamps ("2m ago").

## Data & State

**Discovery pipeline (every tick, ~2s):**
1. `tree --all --json` (1 call) → focused surface UUID + all surfaces with titles.
2. Focused surface only: `sessions list --surface <uuid> --all --json` (1 call) → best omp record (ranking above).
3. Session file via unique-id glob; fingerprint it.
4. Load state; render; decide enqueues. Two execs per tick — matches POC cost.

**Fingerprint (change detection):** `stat` mtime+size of the `.jsonl` + hash of the last 4 KiB (catches same-size appends). Store `fingerprint` + `generated_at` per session in state. If fingerprint matches state → no AI call. Session-id rotation (user restarts omp in the tab) creates a new fingerprint → one enqueue.

**State file:** `<session-cwd>/.omp/summary.json` — colocated per project, matches "state in .omp/summary.json"; global fallback `~/.omp/summary.json` for private-tmp sessions (no writable project). `--state <path>` override flag. Atomic writes (tmp + rename). The TUI on load reads the focused tab's state file → instant paint, then updates.

```json
{
  "version": 1,
  "session_id": "01a0b222-…",
  "surface_id": "126F4FEF-…",
  "fingerprint": "sha256:…",
  "generated_at": "2026-09-18T23:50:00Z",
  "model": "aihub/glm-5.3-flash",
  "last_command": { "raw": "go", "goal": "fold Q1 into plan and push" },
  "summary": "Phase 1 complete and pushed…",
  "ai_expects": "decision on prod domain",
  "ai_last_message": "Q1 folded in (→ 4b5ff2a on main)…"   // full text, for copy
}
```

**Tab rows:** title from `tree`; per-tab freshness from the state files we can reach (lookup by surface_id→session_id recorded at last generation). Rows for tabs with no omp session show "no omp" (not clickable-focusable? still focusable — it's a shell tab; just no summary).

## AI Summarizer

- **Input** (from session JSONL tail): tab title; last user message (full, ≤2k chars); last assistant message with `stopReason != toolUse` text (≤2k); last ~8 tool call `name/intent` lines (the "last command" trail); turn counts. Truncate deterministically.
- **Output**: strict JSON `{goal, summary, ai_expects, ai_last_message}` — one short paragraph each, ≤ 240 chars for display fields, full text for `ai_last_message`. Lenient parse (strip code fences, find first `{`…last `}`); on parse failure → 1 retry, then keep previous summary + error badge.
- **Client**: `ai.NewClientWith(aiOverride{Model: flag})` — the failover chain picks provider/keys from cly config; `--model` overrides. Record resolved model (or configured model) into state → model badge. Show "ai: <model>" always (user req #6).
- **Queue/worker**: bounded channel (cap 8, drop-oldest per session: new fingerprint for same session replaces queued older one), 2 workers default (`--workers`). No queue growth from rapid tab switching — coalescing is per-session-id. Errors → exponential backoff, max 2 attempts per fingerprint, then badge "failed (press r)".

## Architecture

```
modules/omp/              ← new utility module (self-contained)
  cmd.go                  Register(parent): omp → summary subcommand; flags: --interval, --workers, --model, --once, --state
  summary.go              Bubbletea v2 model/update/view; mouse; sections; hit-testing map per render
  discover.go             ticker → pkg/cmux.Tree + SessionsList → focused session record + all tabs
  session.go              JSONL tail reader → fingerprint + summarizer input; glob by session id
  summarize.go            queue + worker pool + prompt + JSON parse (uses pkg/ai)
  state.go                summary.json load/save (atomic), per-cwd path resolution
pkg/cmux/cmux.go          extend: Tree(ctx), SessionsList(ctx, surface), FocusSurface(ctx, surface) — JSON structs for --id-format both
```

No cross-module imports. Shared styles via `pkg/style` if they exist, else local.

## Steps

1. **`pkg/cmux` extensions** — add `Tree` (with `--id-format both`), `SessionsList`, `FocusSurface`; typed structs; unit tests with a fake command runner (interface already implied by `BinaryAvailable` pattern); verify against the live socket once (`go run` snippet). *Verify: tests green; live tree returns focused surface UUID matching `cmux identify`.*
2. **Module skeleton + frame** — `modules/omp/` with static Charm v2 frame rendering the four sections + footer from hardcoded data; alt-screen off (sidebar pane should own its scrollback? **No: full-redraw inline view** like the POC — sidebar panes should not alt-screen; use tea's inline rendering). *Verify: `go build`, `cly omp summary` renders in a 53-col pane; `q` quits.*
3. **Discovery + instant paint** — wire discover.go + state.go; on start read focused tab's state file and render it; ticker refreshes focus/titles; no AI yet. *Verify: switch tabs in cmux → title/status line follows within one tick.*
4. **Summarizer pipeline** — fingerprinting, tail reader, queue/workers, prompt, parse, state write, model badge. *Verify: fingerprint-only regeneration (edit nothing → no AI calls; append one message → exactly one AI call); 3 quick tab switches → 3 queued updates, ≤2 running concurrently; state file written atomically; model name displayed.*
5. **Clickability + polish** — mouse hit-testing for tab rows/copy/section collapse; clipboard; spinner/badges/relative time; pause; `--once` non-TUI mode (prints one summary + exits, keeps scripting parity with the POC). *Verify: click row → cmux focus changes (observe `cmux tree`); copy → paste works; collapsed sections persist per session (in-memory only).*
6. **Cleanup + register** — wire `modules/omp.Register(RootCmd)` in `cmd/root.go`; `go mod tidy`; remove nothing else; update cly CHANGELOG (ag:changelog). *Verify: `go build ./...`, `go vet ./modules/omp/...`, full `go test ./...`, help shows `cly omp summary`.*

## Verification (whole feature)

- `go build ./... && go test ./modules/omp/... ./pkg/cmux/...` — unit tests for fingerprint, tail parser (fixture JSONL), queue coalescing, worker cap, state round-trip, lenient JSON parse, click hit-test.
- Live dogfood: run `cly omp summary` pinned in the actual cmux sidebar pane; work in 3 omp tabs; confirm instant-paint from cache, async fill-in, no regen on idle, click-to-focus round-trip, copy action.
- Kill mid-generation → restart → stale state renders, queue self-heals.

## Out of Scope

- Non-omp agents (claude/codex/gemini tabs) — later; the tab list architecture allows it.
- Native cmux custom sidebar (`~/.config/cmux/sidebars/`) — this ships as a TUI in a pinned terminal pane (what the screenshot shows); port later if wanted.
- Editing/creating omp sessions, history browsing beyond last message, per-section AI config.

## Risks / Known Gotchas

- `cmux sessions list --surface` **requires UUID** (refs silently no-match) — always pass `.active.surface_id` from `--id-format both`.
- jaq/bash POC bugs don't apply (native Go now), but the JSONL shapes (epoch-ms timestamps, `stopReason: toolUse`, `toolCall.intent`) carry over.
- Reused surfaces have 600+ hook records — ranking is mandatory, not optional.
- Large sessions (7 MB) — never read whole file; tail only (last ~64 KiB is plenty).
- `.omp/summary.json` in projects → add `.omp/` to global gitignore (`~/.config/git/ignore`) to avoid repo churn; do not commit.
- AI latency (~seconds) → UI must never block on it; every render path reads state only.
