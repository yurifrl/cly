---
created: 2026-08-16
project: cly
description: Implemented shared location-aware AI model failover for CLY requests and git-commits.
session_id: 01a00bdc-2a86-7f4e-bb66-ddc8cb0d17da
resume_with: cly agent-session resume --provider pi location-aware-ai-failover
checkpoint_file: .agents/checkpoints/location-aware-ai-failover.md
---

## Context
User requested generic provider failover: retain location-based primary selection, then try all configured models once on any failure, including models normally restricted to another directory.

## Decisions
- `pkg/ai` owns candidate ordering; commands do not know provider names.
- Order is selected location match, other matching entries by weight, default entries, then all remaining configured entries.
- Each request starts the complete chain anew; no session stickiness or retry cycling.
- Any failure triggers the next candidate. Exhaustion returns one combined error.
- Existing Dotfiles config already declares the two entries: NSX-directory conditional primary and default fallback; no config schema or hardcoded provider names added.

## Current State
- Added `pkg/llm/fallback.go` with `FallbackClient` for `Complete` and `Stream`.
- `pkg/ai.NewClientWith` constructs the ordered chain and logs failovers to stderr.
- `modules/git-commits` now calls shared `ai.NewClientWith`, so `git wip` receives failover.
- Added fallback and ordering tests.

## Lessons
- Pi retry settings retry a single provider only; cross-provider resilience belongs in CLY's shared AI client construction.
- `git wip` failure originated in CLY `git-commits`, not Pi's `/git wip` extension command.

## Next Steps
- Optional: manually run `git wip` with an intentionally unavailable primary to observe stderr failover against live credentials.
- Repository-wide `go test ./...` still has unrelated failures in the pre-existing dirty `modules/dotfiles` area; do not modify those files without resolving ownership.
