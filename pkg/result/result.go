// Package result provides a generic three-state value type for any
// operation that can yield a value, yield nothing, or fail.
//
// # Why three states
//
// Go's idiomatic (T, error) return collapses two distinct outcomes into
// one channel: "missing but fine" and "broken." That is fine for
// imperative actions (write, send, RPC) where absence is meaningless.
// It is awkward for lookups, parses, and optional reads where "not
// present" is a normal, non-error outcome that a caller should be able
// to distinguish from a real failure.
//
// Result[T] makes the third state explicit:
//
//   Ok    — value is present and valid
//   Empty — no value; not an error, just absent
//   Error — value was attempted but invalid (or required and missing)
//
// # When to use
//
//   ✓ Lookups (env vars, store/cache reads, config keys)
//   ✓ Parses (string -> typed value, where invalid input is meaningful)
//   ✓ Optional fetches (HTTP 200 vs 404 vs 5xx)
//
//   ✗ Imperative actions (writes, sends, RPCs) — keep returning error.
//     A successful write is not Ok(void); it is just nil.
//
// # Invariants (enforced by constructors)
//
//   IsOk()    ⇔ value is present, err == nil
//   IsEmpty() ⇔ no value, err == nil
//   IsError() ⇔ err != nil
//
// Exactly one predicate is true for any Result. Error(nil) is coerced
// to Empty so the IsError ⇔ err != nil invariant always holds.
//
// # Why constructors require [T] for Empty and Error
//
// Go infers type parameters only from argument types. Empty has no
// arguments. Error's only argument is the universal error interface,
// which carries no T information. Both must be written explicitly:
//
//   result.Ok("foo")              // T inferred -> Result[string]
//   result.Empty[string]()        // T explicit
//   result.Error[string](io.EOF)  // T explicit
//
// This is a Go limitation, not a design choice. It only appears at
// internal helper boundaries; user-facing call sites rarely write [T].
package result

import "errors"

// state is the discriminator for Result's three variants. Unexported
// to keep zero-value Result behavior controlled (zero value is Empty).
type state uint8

const (
	stateEmpty state = iota // zero value of Result is Empty
	stateOk
	stateError
)

// Result is a three-state value: Ok(T), Empty, or Error(error).
//
// The zero value of Result[T] is a valid Empty[T]. This means
// `var r Result[string]` is safe to use without initialization.
//
// Result is a value type — copies are cheap and independent.
type Result[T any] struct {
	s   state
	val T
	err error
}

// -----------------------------------------------------------------------------
// Constructors
// -----------------------------------------------------------------------------

// Ok returns a Result holding v. T is inferred from v's type.
func Ok[T any](v T) Result[T] {
	return Result[T]{s: stateOk, val: v}
}

// Empty returns a Result with no value and no error.
//
// T must be specified explicitly because Go cannot infer it without
// an argument:
//
//   r := result.Empty[string]()
func Empty[T any]() Result[T] {
	return Result[T]{s: stateEmpty}
}

// Error returns a Result in the Error state carrying err.
//
// As a safety guarantee, Error(nil) is coerced to Empty so the
// IsError ⇔ err != nil invariant always holds. Callers should pass a
// non-nil error; passing nil is treated as a programmer hint that
// "there is nothing here," not "there was an error."
//
// T must be specified explicitly because err carries no T information:
//
//   r := result.Error[string](io.EOF)
func Error[T any](err error) Result[T] {
	if err == nil {
		return Empty[T]()
	}
	return Result[T]{s: stateError, err: err}
}

// -----------------------------------------------------------------------------
// Adapters: bridge Go's native (T, error) idiom into Result
// -----------------------------------------------------------------------------

// From converts an idiomatic Go (T, error) return into a Result.
// If err is non-nil, the result is Error(err). Otherwise it is Ok(v).
//
// From treats `err != nil` as authoritative — v is ignored when err is
// set, mirroring Go's convention that a non-nil error means the value
// is meaningless.
//
// From never produces Empty. Callers that need to distinguish an
// absent value from a zero value must use FromOpt or FromPtr.
func From[T any](v T, err error) Result[T] {
	if err != nil {
		return Error[T](err)
	}
	return Ok(v)
}

// FromOpt converts an idiomatic Go (T, ok bool) return into a Result.
// If ok is false, the result is Empty. Otherwise it is Ok(v).
//
// Useful for adapting map lookups, type assertions, and sync.Map.Load.
func FromOpt[T any](v T, ok bool) Result[T] {
	if !ok {
		return Empty[T]()
	}
	return Ok(v)
}

// FromPtr converts a *T into a Result. A nil pointer becomes Empty.
// A non-nil pointer becomes Ok(*p).
//
// Useful for adapting APIs that return pointers to signal optionality.
func FromPtr[T any](p *T) Result[T] {
	if p == nil {
		return Empty[T]()
	}
	return Ok(*p)
}

// -----------------------------------------------------------------------------
// State predicates — exactly one is true for any Result
// -----------------------------------------------------------------------------

// IsOk reports whether r holds a value.
func (r Result[T]) IsOk() bool { return r.s == stateOk }

// IsEmpty reports whether r is Empty (no value, no error).
func (r Result[T]) IsEmpty() bool { return r.s == stateEmpty }

// IsError reports whether r holds an error.
func (r Result[T]) IsError() bool { return r.s == stateError }

// -----------------------------------------------------------------------------
// Always-safe direct accessors — never panic
// -----------------------------------------------------------------------------

// Error returns the underlying error, or nil if r is Ok or Empty.
//
// This is the canonical way to test for the Error state:
//
//   if err := r.Error(); err != nil { ... }
//
// Note: Result does NOT implement Go's error interface. Go's interface
// requires Error() string; this method returns error. The mismatch is
// intentional — a Result is not an error, it may merely contain one.
func (r Result[T]) Error() error { return r.err }

// Empty reports whether r is in the Empty state. It is an alias for
// IsEmpty kept for readability in conditional expressions:
//
//   if r.Empty() { ... }
func (r Result[T]) Empty() bool { return r.s == stateEmpty }

// -----------------------------------------------------------------------------
// Lossless destructure
// -----------------------------------------------------------------------------

// Unwrap destructures r into the native Go (T, error) shape:
//
//   Ok(v)    -> (v,    nil)
//   Empty    -> (zero, nil)
//   Error(e) -> (zero, e)
//
// Empty and Ok-holding-the-zero-value are indistinguishable from
// Unwrap alone. Callers that need to tell them apart use Empty()
// alongside:
//
//   val, err := r.Unwrap()
//   if err != nil { return err }
//   if r.Empty()  { val = fallback() }
func (r Result[T]) Unwrap() (T, error) {
	if r.s == stateOk {
		return r.val, nil
	}
	var zero T
	return zero, r.err
}

// -----------------------------------------------------------------------------
// Convenience views — collapse states to a single value
// -----------------------------------------------------------------------------

// Or returns the value when r is Ok, otherwise def. Both Empty and
// Error fall through to def — Or is the "I just want a value, default
// for anything else" accessor.
func (r Result[T]) Or(def T) T {
	if r.s == stateOk {
		return r.val
	}
	return def
}

// OrElse returns the value when r is Ok, otherwise the result of fn.
// Use when computing the default is expensive or has side effects you
// only want to pay for in the non-Ok path.
func (r Result[T]) OrElse(fn func() T) T {
	if r.s == stateOk {
		return r.val
	}
	return fn()
}

// Must returns the value when r is Ok, and panics otherwise.
//
// Reserved for init code, tests, and other contexts where an absent
// or invalid value is a programmer bug rather than a runtime
// condition. The panic value is the underlying error for the Error
// state, or ErrEmpty for the Empty state.
//
// Do not use Must in normal request-path code. Use Or, OrElse, or
// Unwrap instead.
func (r Result[T]) Must() T {
	switch r.s {
	case stateOk:
		return r.val
	case stateEmpty:
		panic(ErrEmpty)
	default:
		panic(r.err)
	}
}

// -----------------------------------------------------------------------------
// Branching
// -----------------------------------------------------------------------------

// Match dispatches to exactly one of the three handlers based on r's
// state. Any handler may be nil; nil handlers are treated as no-ops.
//
//   r.Match(
//       func(v T)        { use(v) },
//       func()           { useDefault() },
//       func(e error)    { logErr(e) },
//   )
//
// Match is a convention, not a compile-time guarantee — Go has no
// exhaustiveness check. Prefer Match over hand-rolled switch chains
// when readability matters; prefer plain if/else when only one branch
// is interesting.
func (r Result[T]) Match(onOk func(T), onEmpty func(), onError func(error)) {
	switch r.s {
	case stateOk:
		if onOk != nil {
			onOk(r.val)
		}
	case stateEmpty:
		if onEmpty != nil {
			onEmpty()
		}
	case stateError:
		if onError != nil {
			onError(r.err)
		}
	}
}

// -----------------------------------------------------------------------------
// Sentinels
// -----------------------------------------------------------------------------

// ErrEmpty is the panic value used by Must when called on an Empty
// Result. It is also useful when a caller wants to map Empty to an
// error explicitly:
//
//   val, err := r.Unwrap()
//   if err == nil && r.Empty() {
//       err = result.ErrEmpty
//   }
var ErrEmpty = errors.New("result: empty")
