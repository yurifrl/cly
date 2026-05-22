# `pkg/result` + `pkg/envs` — Design Spec

Status: drafted, not implemented.
Owner: yuri
Decided so far: function-per-var public API, project-wide Result type, `Unwrap` as the only lossless destructure.

---

## Goals

1. Single project-wide three-state value type — `result.Result[T]` — for anything where "missing" is a real third state alongside Ok/Err (env vars, store reads, config lookups, optional fetches).
2. Centralize every env-var literal string in one package — `pkg/envs` — exposed as **typed function accessors**, never as keys/constants.
3. Clean separation: `pkg/result` knows nothing about env vars; `pkg/envs` is a thin domain layer on top.

---

## `pkg/result`

### Layout

```
pkg/result/
├── result.go        Result[T] type + constructors + methods
├── helpers.go       From, FromOpt, FromPtr adapters
└── result_test.go
```

### Type

```go
package result

type state uint8

const (
    stateEmpty state = iota
    stateOk
    stateError
)

// Result[T] is a three-state value:
//   Ok    — value present
//   Empty — absent, but not an error
//   Error — present but invalid, OR required and missing
type Result[T any] struct {
    s   state
    val T
    err error
}
```

### Public constructors

```go
func Ok[T any](v T) Result[T]
func Empty[T any]() Result[T]
func Error[T any](err error) Result[T]
```

### Adapters (Go-native → Result)

```go
func From[T any](v T, err error) Result[T]   // err != nil -> Err, else Ok
func FromOpt[T any](v T, ok bool) Result[T]  // !ok -> Empty, else Ok
func FromPtr[T any](p *T) Result[T]          // nil -> Empty, else Ok(*p)
```

### Method roster (final)

```go
// State predicates
IsOk()    bool
IsEmpty() bool
IsError() bool

// Direct, always-safe accessors
Error() error    // nil if Ok or Empty; the error if Errored
Empty() bool     // true iff Empty

// Lossless destructure (2 returns)
Unwrap() (T, error)
//   Ok(v)    -> (v,    nil)
//   Empty    -> (zero, nil)
//   Error(e) -> (zero, e)

// Convenience views
Or(def T) T                    // Ok→val, Empty/Error→def
OrElse(fn func() T) T
Must()    T                    // panics on Empty or Error

// Branching
Match(onOk func(T), onEmpty func(), onError func(error))
```

`Unwrap` returns native Go `(T, error)` shape; `Empty()` disambiguates Empty from Ok-with-zero when needed. `Error()` and `Empty()` never panic. `Must()` is the only panicking accessor.

### Sentinel errors

```go
var ErrEmpty = errors.New("result: empty")  // only if a caller maps Empty -> error manually
```

### Scope discipline

- ✅ Use `Result[T]` for **lookups, parses, optional reads** — anywhere "missing" is meaningful.
- ❌ Don't use `Result[T]` for **imperative actions** (writes, sends, RPCs). They stay `error`.

---

## `pkg/envs`

### Layout

```
pkg/envs/
├── session.go       SessionName / SetSessionName / HasSessionName / UnsetSessionName
├── cmux.go          CmuxSurfaceID, CmuxTabID
├── zellij.go        Zellij, InZellij, ZellijSession
├── claude.go        ClaudeVerbose
├── notify.go        Sound
├── source.go        Source interface + osSource + Use(Source)
├── errors.go        ErrInvalid, *Error type
├── internal.go      Private helpers: readString, readBool, write, clear
└── *_test.go
```

### Public API — function per env var

```go
// session.go
func SessionName() result.Result[string]
func SetSessionName(v string) error
func HasSessionName() bool
func UnsetSessionName()

// cmux.go
func CmuxSurfaceID() result.Result[string]
func CmuxTabID() result.Result[string]

// zellij.go
func Zellij() result.Result[string]
func InZellij() bool
func ZellijSession() result.Result[string]

// claude.go
func ClaudeVerbose() result.Result[bool]

// notify.go
func Sound() result.Result[string]
```

No `Key` type. No registry map. No `Get[T]` / `Set[T]` generic accessors. No exported constants.

Setters return plain `error` (writes are imperative, not lookups).

### Internals (private helpers)

```go
// internal.go
func readString(name string, fallbacks ...string) result.Result[string]
func readBool(name string, fallbacks ...string)   result.Result[bool]
func write(value string, names ...string) error
func clear(names ...string)
func has(name string, fallbacks ...string) bool
```

These take literal strings (env var names), not "keys."

### Example file

```go
// session.go
package envs

import "github.com/yurifrl/cly/pkg/result"

func SessionName() result.Result[string] {
    return readString("CLY_SESSION_NAME", "CLAUDE_SESSION_NAME")
}

func SetSessionName(v string) error {
    return write(v, "CLY_SESSION_NAME", "CLAUDE_SESSION_NAME")
}

func HasSessionName() bool {
    return has("CLY_SESSION_NAME", "CLAUDE_SESSION_NAME")
}

func UnsetSessionName() {
    clear("CLY_SESSION_NAME", "CLAUDE_SESSION_NAME")
}
```

The literal `"CLY_SESSION_NAME"` lives in **exactly one place** in the codebase: this file.

### Resolution order (inside `read*`)

1. canonical key set & non-empty → parse → `Ok` / `Err`
2. each fallback in order → parse → `Ok` / `Err`
3. nothing found → `Empty`

`write` writes canonical AND every alias. `clear` removes canonical AND every alias.

### Source interface (testability)

```go
// source.go
type Source interface {
    Lookup(name string) (string, bool)
    Set(name, value string) error
    Unset(name string)
}

func Use(s Source)        // tests inject MapSource; default is osSource
```

### Errors

```go
// errors.go
var ErrInvalid = errors.New("envs: invalid value")

type Error struct {
    Name  string
    Cause error
    Value string  // raw input that failed to parse
}
func (e *Error) Error() string
func (e *Error) Unwrap() error
```

Used inside `Result.Err()` when parsing fails.

---

## Call site examples

### Simple default
```go
name := envs.SessionName().Or("anonymous")
```

### Distinguish all three states
```go
r := envs.SessionName()
if err := r.Error(); err != nil {
    log.Warnf("bad session name: %v", err)
}
val, _ := r.Unwrap()
if r.Empty() {
    val = generateName()
}
use(val)
```

### Pattern match
```go
envs.SessionName().Match(
    func(v string) { tab.Rename(v) },
    func()         { tab.Rename(generated()) },
    func(e error)  { log.Errorf("invalid: %v", e) },
)
```

### Required at boot
```go
val, err := envs.ClaudeVerbose().Unwrap()
if err != nil { return err }
if envs.ClaudeVerbose().Empty() { return errors.New("CLAUDE_VERBOSE not set") }
```

### Setters
```go
if err := envs.SetSessionName("my-feature"); err != nil { ... }
```

---

## Migration plan

1. **Land `pkg/result`** — type, constructors, methods, full tests. Zero callers initially.
2. **Land `pkg/envs`** — built on `pkg/result`. Cover the env vars currently used:
   - `CLY_SESSION_NAME` (+ alias `CLAUDE_SESSION_NAME`)
   - `CMUX_SURFACE_ID`, `CMUX_TAB_ID`
   - `ZELLIJ`, `ZELLIJ_SESSION_NAME`
   - `CLAUDE_VERBOSE`
   - `SOUND`
3. **Migrate existing call sites**:
   - `pkg/session/session.go` → `envs.SessionName()` / `envs.SetSessionName(...)`
   - `modules/piwrap/piwrap.go` → `envs.SetSessionName(name)` (replaces dual-write)
   - `modules/notify/debug.go` → loop over `envs.*()` accessors instead of hardcoded `os.Getenv`
   - `pkg/notify/zellij.go` + `modules/zs/session.go` → `envs.Zellij()` / `envs.ZellijSession()`
4. **Adopt opportunistically** elsewhere — never a big-bang rewrite. Anywhere a `(T, bool, error)` shape appears in the codebase, consider migrating to `result.Result[T]`.

---

## Locked decisions

- **Naming**: `Ok` / `Empty` / `Error`. Predicates `IsOk` / `IsEmpty` / `IsError`. Constructors `result.Ok` / `result.Empty` / `result.Error`. State enum `stateOk` / `stateEmpty` / `stateError`.
- **`Error()` accessor** — always safe, returns `nil` when not in Error state.
- **`Empty()` accessor** — always safe, returns bool. Mirrors `IsEmpty()`; both kept for ergonomics.
- **`Unwrap()` returns `(T, error)`** — native Go shape. Empty maps to `(zero, nil)`. `Empty()` disambiguates when needed.
- **No `Value()` accessor** — `Unwrap` is the sole value getter.
- **`Or(def T)`** swallows both Empty and Error. Strict callers use `Unwrap` + `Empty` or `Must`.
- **No `Map` / `AndThen`** — skipped until a real call site needs chaining.
- **`Must()`** kept — only panicking accessor, bounded and well-named.
- **`From(v, err)`** — `err != nil` wins → `Error(err)`, else `Ok(v)`. Mirrors Go's `(T, error)` convention.
- **Result's Error state holds plain Go `error`** — no custom wrapper type. `errors.New`, `fmt.Errorf`, custom error structs all work.

---

## What this design deliberately does NOT have

- No `Key` type, no `keys.go`, no env-var registry map.
- No `envs.Get[T](Key)` / `envs.Set[T](Key, v)` generic accessors.
- No `envs.All()` / iteration over all known env vars (if needed, write a dedicated function in `pkg/envs` that returns a struct of accessor results — don't expose iteration as a primitive).
- No magic struct binding (Viper already does that; this is lower-level).
- No package-level mutable config beyond the swappable `Source`.
- `Unwrap()` is the only lossless destructure on `Result[T]`.
