# Remake: omp summary as cmux custom sidebar card

Goal: a vertical single-card cmux sidebar for the **active surface only** — surface name,
last command (goal + what I asked), AI summary (what the AI is asking/expecting), and the
summarizer model badge. Updates run async via a daemon (queue + parallel workers), cached
in `.omp/summary.json`, no AI regeneration when nothing changed. Replaces the all-sessions
watch TUI.

## Context — wrong turn being fixed

- Delivered: Bubbletea grid TUI (`cly omp y summary`) listing ALL omp sessions; only usable
  as watch mode. User wanted ONE active-surface card living inside the cmux sidebar.
- Sidebar reality (verified): sidebar files can't read files, run processes, or fetch
  network. They bind to live cmux state only. The one writable per-workspace text slot the
  sidebar can read is `description` — set with
  `cmux workspace-action --action set-description --description <text> --workspace <id>`.
- Sidebar subset (verified): `.split(separator:)`, `.hasPrefix`, user `func`, monospaced
  fonts, capsules, status dots all supported in `~/.config/cmux/sidebars/<name>.swift`.
  Context also natively has `workspaces[i].agents[]` (`status`: idle|working|needs_input|ended)
  and `selectedTitle` — status dot needs no daemon.

## Design decisions

- **Transport = workspace `description`.** Daemon writes a tagged-line card there; the
  sidebar parses and styles it. Side effect: description becomes tool-managed for
  workspaces the daemon touches (acceptable; it's our dedicated use).
- **One card = active surface.** Switching tabs enqueues an update per visited tab (3 tabs
  → 3 enqueued updates, run in parallel). Returning to a tab shows its cached summary
  instantly; refresh is background. Fingerprint = transcript size:mtime + surface id;
  unchanged ⇒ skip AI entirely.
- **Race guard:** a worker that finishes after the user switched away pushes only if its
  session is still the active surface; otherwise cache-only (description of the shared
  workspace isn't clobbered by a stale tab).
- **Renderer = `.swift`** custom sidebar `omp-summary`. Fallback `.js` if
  `cmux sidebar validate` flags gaps.
- **Command surface:** `cly omp summary` → daemon (user's literal ask). `--once` prints the
  current card from state and exits (non-watch access). `omp y summary` grid TUI is deleted;
  `omp y` stays for `extensions install`.
- **Autostart:** manual for now (`o summary`); optional cmux automation later.

## Steps

1. **State v2** — `modules/omp/summary/state.go`: per-session entry becomes
   `{surface{id,title,workspace}, last_command{text,exit,at}, user_ask{text,at},
   goal{text}, ai_status{text}, model, provider, fingerprint, updated_at_unix, turns}`.
   Load v1 files as stale (regenerate on next fingerprint mismatch); keep atomic save.
2. **Active-target resolver** — `discover.go`: `ActiveTarget()` = `pkg/cmux.RunTree` →
   active surface id → `cmux sessions list --json` match (surface_id / active flags /
   cwd) → transcript via existing `transcriptPath`. All-session enumeration dropped from
   the daemon path (fixtures stay for tests).
3. **Description pusher** — new `push.go` + `pkg/cmux` helper: format tagged lines
   (`▸ <cmd> (exit N)`, `? <asked>`, `◆ <goal>`, `● <ai status>`, `✦ <model>`,
   `⏱ <updated epoch>`), run `cmux workspace-action set-description`; race-guard by
   re-checking active surface at push time; cap card at ~400 chars.
4. **Engine rewrite** — `engine.go`: poll tree @1s; on active-session change: instantly
   push the cached card for that session's project state, enqueue when fingerprint
   differs. Keep queue + N=3 parallel workers; one AI call per update → SaveState → push
   (guarded).
5. **Summarizer prompt v2** — `summarizer.go`: inputs = surface title, last user message,
   last command + exit, tail of last assistant message; output JSON `{"goal","ai_status"}`;
   record model/provider. Reuse `pkg/ai` client, config, worker pool.
6. **Command rewiring** — `modules/omp/cmd.go`: register `summary` directly under `omp`
   (daemon by default; flags `--once`, `--verbose`). Delete `y summary` registration,
   `tui.go` grid + Wake model wiring and all TUI-only symbols. `pkg/cmux` Tree/SessionsList
   stay (used by resolver/tests).
7. **Sidebar card** — `~/.config/cmux/sidebars/omp-summary.swift`: vertical card for the
   selected workspace: surface name (`selectedTitle`) + status dot from `w.agents`;
   section "Last command" (monospaced ▸ line, "Asked", "Goal"); divider; section "AI"
   (● line); model capsule (✦ line); "updated Xm ago" (⏱ line vs `clock.epoch`).
   `cmux sidebar validate omp-summary` then select/open it.
8. **Ship** — unit tests (resolver from tree fixture, fingerprint skip, race guard, v1→v2
   load), build, CHANGELOG entry, commit, push, `graphify update .`.

## Verification

- `go test ./modules/omp/summary ./pkg/cmux` — fixture-driven resolver, state migration,
  fingerprint skip, race-guard tests pass.
- Daemon dogfood: run `cly omp summary --verbose` in a pane; switch 3 tabs → verbose log
  shows 3 enqueues running in parallel; `cmux workspace list --json` shows card
  descriptions; switching back to an unchanged tab shows no new AI call (skip logged).
- Sidebar: `cmux sidebar validate omp-summary`; open in the sidebar; verify with
  screenshot/AX: vertical card, active surface name, both sections, model pill; switching
  workspace moves the card.
- E2E: a real AI round-trip writes v2 `.omp/summary.json` in the project cwd and the card
  shows the fresh goal/ai-status lines.

## Out of scope

- Showing summaries for multiple tabs simultaneously (card is active-only, per request).
- Changes to omp session format or cmux itself; daemon autostart via cmux automation.
