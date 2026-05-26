# Native macOS Notifications for `cly`
A signed Swift notifier daemon embedded into `pkg/notify`, delivering modern macOS notifications with action buttons (snooze, retry) that feed back into the `every` loop, with no codesigning required of consuming Go binaries.

> Note: This is a draft to organize ideas and scope before implementation.

## Goal
Replace the legacy `terminal-notifier` shell-out in `cly every` with a native macOS notification subsystem that:
- Supports modern UX (action buttons, persistent banners, swipe-to-dismiss in Notification Center)
- Lets users **snooze** a failing task for 5 minutes from the notification itself
- Lets users **retry** a gave-up task from the notification itself
- Works with vanilla `go build` on the consumer side (no codesign requirement on `cly`)
- Lives in `pkg/notify` as a standalone-flavor package (extractable later as its own Go module)

## Architecture
A signed `cly-notifier.app` bundle is embedded into the Go binary via `go:embed`, extracted on first run to `~/Library/Application Support/cly/`, and spawned as a long-lived daemon. Go and the Swift daemon talk newline-delimited JSON over a Unix socket. The daemon owns the bundle ID and the macOS notification permission; the consuming Go binary stays unsigned.

```
cly every (Go)                          cly-notifier.app (Swift, signed)
     │                                          │
     │ ──exec --socket /tmp/cly-X.sock─────────►│  UNUserNotificationCenter
     │                                          │  requestAuthorization()
     │ ──{op:send,group,title,actions:[...]}───►│
     │                                          │  schedule notification
     │                                          │
     │                                          │  user clicks "Snooze 5m"
     │ ◄──{op:action,group:cly.every.X,id:snooze}
     │                                          │
     │ loop: SnoozeUntil = now+5m for task X    │
     │ parent exit → SIGTERM ──────────────────►│  exit
```

## Components

### `pkg/notify` (standalone flavor)
- `types.go` — `Notification` (+ `Actions []Action`), `Action{ID,Title}`, `ActionEvent{Group,ActionID}`
- `notifier.go` — `Notifier` interface (+ `Events() <-chan ActionEvent`), `MultiNotifier` with event fan-in
- `beeep_notifier.go` / `zellij.go` — existing backends, return nil-safe closed event channel
- `native_darwin.go` — `NativeMacOSNotifier`: bundle extract, codesign verify, daemon spawn, socket I/O
- `native_other.go` — stub returning `Available()=false`
- `embed_darwin.go` / `embed_other.go` — `//go:embed assets/cly-notifier.app.tar.gz` (darwin only)
- `generate.go` — `//go:generate ./swift/build.sh`
- `swift/Sources/{main,Notifier,Categories,Socket}.swift` — daemon source
- `swift/Info.plist` — bundle ID `dev.yurifrl.cly.notifier`
- `swift/build.sh` — swiftc arm64 + x86_64 → lipo → bundle assembly → codesign

### `modules/every`
- `state.go` — adds `SnoozeUntil time.Time` per-task field
- `loop.go` — checks `SnoozeUntil` before run; goroutine reading `notifier.Events()`; maps `id=snooze` → `SnoozeUntil = now+5m`, `id=retry` → reset retry counter + force next tick
- `notify/notify.go` — kept as thin shim with same `Send(ctx, Notification)` signature; delegates to `pkg/notify`; maps level → default `Actions`

## Key Decisions

**Backend selection — no user config**
On darwin, `pkg/notify.New()` tries native first (extract bundle + codesign verify + daemon ping). On any failure, single-line stderr warning and fall back to `beeep`. Linux/Windows always `beeep`.

**Signing model — bundle signed, consumer not**
The embedded `cly-notifier.app` is signed at build time with a Developer ID Application certificate. macOS notification permission is granted to the bundle's ID, independent of the calling Go binary. Consumers run `go build` and ship unsigned binaries.

**Build trigger — `go generate`**
Not `go build`. Fresh checkout on darwin runs `go generate ./pkg/notify/...` once; CI runs it before `go build`. A 1-byte placeholder tarball is committed so the embed compiles even before `go generate` runs; native notifier detects placeholder and falls back.

**Codesign secret — 1Password reference**
`build.sh` reads `CLY_NOTIFIER_SIGN_OP_REF` (e.g. `op://Personal/Apple-Developer-ID/identity`) and resolves via `op read`. Falls back to `CLY_NOTIFIER_SIGN_ID` literal env, or ad-hoc `-` for local dev (works on the developer's own Mac, not distributable).

**Default actions per `every` level**
- `failing` → `[Snooze 5m] [Dismiss]`
- `recovered` → `[]` (auto-dismiss banner, still persists in Notification Center)
- `gaveup` → `[Retry] [Dismiss]`

**Snooze semantics — module-defined, suppress run**
`pkg/notify` only ferries action IDs back. `every` interprets `snooze` as "skip runs for 5min" by writing `SnoozeUntil` into per-task state. Loop keeps running but the snoozed task is skipped. Other modules can interpret the same action differently when they consume `pkg/notify`.

**Wire protocol — newline JSON over Unix socket**
- Inbound: `{"op":"send","group":"cly.every.foo","title":"...","body":"...","sound":"Basso","actions":[{"id":"snooze","title":"Snooze 5m"}]}`
- Outbound: `{"op":"action","group":"cly.every.foo","id":"snooze"}`
- Socket path: `~/Library/Application Support/cly/notifier.<ppid>.sock` (multiple `cly` runs don't collide)

**Standalone "flavor", not enforced**
Single `go.mod`. Isolation is policy: `pkg/notify` imports only stdlib + beeep; `modules/every` imports only stdlib + `pkg/notify`. A `task lint:isolation` grep target enforces it. Future extraction is a `git filter-repo` away with zero code changes.

**Bundle install path**
`~/Library/Application Support/cly/cly-notifier-v<sha>.app`, hash of the embedded tarball. Old versions garbage-collected on startup. Version-stamped path ensures cleanly upgrading without permission re-prompts.

## Isolation Contract

`pkg/notify` may import:
- stdlib
- `github.com/gen2brain/beeep`

`modules/every` may import:
- stdlib
- `github.com/yurifrl/cly/pkg/notify`

Anything else under `github.com/yurifrl/cly/...` is forbidden. Enforced by `task lint:isolation`.

## Out of Scope
- Migrating other modules to `pkg/notify`
- Notarization (Developer ID signature is enough; notarize when distributing publicly)
- Cross-task snooze coordination
- Configurable snooze durations (fixed 5min for v1)
- Linux/Windows action button support
- Modern `UNUserNotificationCenter` features beyond title/body/sound/actions/group (no rich attachments, no scheduling, no threading beyond `group`)

## Open Questions
- Is `dev.yurifrl.cly.notifier` the final bundle ID, or should it match a registered App ID in your Apple developer account? (Once shipped to a user, changing it = re-prompt.)
- Should the daemon log to a file in addition to parent stderr passthrough, for debugging permission issues?
- Should snooze duration become configurable per-task (config file) before v1 ships, or wait for real-world feedback?
- Should `gaveup`'s `[Retry]` action also reset the failure counter to zero, or just force one immediate next-tick run with the counter intact?

## Implementation Notes

**Tech stack**
- Go: stdlib + beeep, `go:embed`, `os/exec`, `net` (Unix socket), `encoding/json`
- Swift: `Foundation`, `UserNotifications` framework
- Build: `swiftc`, `lipo`, `codesign --options runtime --timestamp`
- Secret: 1Password CLI (`op read`)

**Permission flow**
First run of the daemon shows the macOS notification permission prompt. User approves once; macOS remembers per bundle ID forever. Daemon reports authorization state on startup; Go logs an actionable hint to stderr if denied.

**Test seams**
- Go: fake daemon (Go subprocess echoing protocol) for `NativeMacOSNotifier` tests
- Swift: protocol layer testable in isolation; UN center mocking via protocol abstraction
- `every`: mock `Notifier` with controllable `Events()` channel for action handling tests

## Beads
Tracked under labels `notify`, `every`, `isolation`, `swift`, `build`, `daemon`, `docs`.

| ID | Title | Blocks on |
|---|---|---|
| `cly-aiq` | notify: remove pkg/envs leak from pkg/notify | — |
| `cly-kss` | every: remove pkg/mut and pkg/style leaks from modules/every | — |
| `cly-nqz` | notify: extend pkg/notify with Action and ActionEvent API | cly-aiq |
| `cly-cuo` | notify: Swift notifier daemon (UNUserNotificationCenter) | cly-nqz |
| `cly-skn` | notify: bundle build pipeline + go generate + embed | cly-cuo |
| `cly-kwx` | notify: NativeMacOSNotifier (Go side, daemon lifecycle, socket) | cly-skn |
| `cly-a7v` | notify: auto-select native on darwin in pkg/notify.New | cly-skn |
| `cly-323` | every: add SnoozeUntil to per-task state | cly-kss |
| `cly-7eq` | every: subscribe to notifier ActionEvents and apply snooze/retry | cly-a7v, cly-323 |
| `cly-bcg` | every: refactor modules/every/notify shim to delegate to pkg/notify | cly-7eq |
| `cly-2vx` | notify: docs for native notification subsystem | cly-bcg |

Ready to start: `cly-aiq`, `cly-kss` (parallel, independent).
