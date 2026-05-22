package result_test

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yurifrl/cly/pkg/result"
)

// -----------------------------------------------------------------------------
// Invariants — exactly one predicate is true for any Result
// -----------------------------------------------------------------------------

func TestInvariants_Ok(t *testing.T) {
	r := result.Ok(42)

	assert.True(t, r.IsOk())
	assert.False(t, r.IsEmpty())
	assert.False(t, r.IsError())
	assert.False(t, r.Empty())
	assert.Nil(t, r.Error())
}

func TestInvariants_Empty(t *testing.T) {
	r := result.Empty[int]()

	assert.False(t, r.IsOk())
	assert.True(t, r.IsEmpty())
	assert.False(t, r.IsError())
	assert.True(t, r.Empty())
	assert.Nil(t, r.Error())
}

func TestInvariants_Error(t *testing.T) {
	want := errors.New("boom")
	r := result.Error[int](want)

	assert.False(t, r.IsOk())
	assert.False(t, r.IsEmpty())
	assert.True(t, r.IsError())
	assert.False(t, r.Empty())
	assert.Same(t, want, r.Error())
}

// Zero value of Result[T] must be a valid Empty.
func TestZeroValue_IsEmpty(t *testing.T) {
	var r result.Result[string]

	assert.True(t, r.IsEmpty())
	assert.False(t, r.IsOk())
	assert.False(t, r.IsError())
	assert.Nil(t, r.Error())
}

// Error(nil) must coerce to Empty so IsError ⇔ err != nil holds.
func TestErrorNil_CoercesToEmpty(t *testing.T) {
	r := result.Error[int](nil)

	assert.True(t, r.IsEmpty())
	assert.False(t, r.IsError())
	assert.Nil(t, r.Error())
}

// -----------------------------------------------------------------------------
// Adapters
// -----------------------------------------------------------------------------

func TestFrom(t *testing.T) {
	t.Run("nil error -> Ok", func(t *testing.T) {
		r := result.From("hi", nil)
		assert.True(t, r.IsOk())
		got, err := r.Unwrap()
		require.NoError(t, err)
		assert.Equal(t, "hi", got)
	})

	t.Run("non-nil error wins -> Error, value ignored", func(t *testing.T) {
		boom := errors.New("boom")
		r := result.From("ignored", boom)
		assert.True(t, r.IsError())
		assert.Same(t, boom, r.Error())
	})

	// From never produces Empty — that's what FromOpt/FromPtr are for.
	t.Run("never produces Empty", func(t *testing.T) {
		r := result.From("", nil)
		assert.True(t, r.IsOk())
		assert.False(t, r.IsEmpty())
	})
}

func TestFromOpt(t *testing.T) {
	t.Run("ok=true -> Ok", func(t *testing.T) {
		r := result.FromOpt(7, true)
		assert.True(t, r.IsOk())
	})

	t.Run("ok=false -> Empty (value ignored)", func(t *testing.T) {
		r := result.FromOpt(7, false)
		assert.True(t, r.IsEmpty())
	})
}

func TestFromPtr(t *testing.T) {
	t.Run("non-nil -> Ok with dereferenced value", func(t *testing.T) {
		v := "hello"
		r := result.FromPtr(&v)
		assert.True(t, r.IsOk())
		got, _ := r.Unwrap()
		assert.Equal(t, "hello", got)
	})

	t.Run("nil -> Empty", func(t *testing.T) {
		r := result.FromPtr[string](nil)
		assert.True(t, r.IsEmpty())
	})

	// FromPtr returns Ok(*p) by value — caller mutating the original
	// pointer must not change the captured value.
	t.Run("captures by value, not by reference", func(t *testing.T) {
		v := 1
		r := result.FromPtr(&v)
		v = 99
		got, _ := r.Unwrap()
		assert.Equal(t, 1, got)
	})
}

// -----------------------------------------------------------------------------
// Unwrap — lossless destructure to (T, error)
// -----------------------------------------------------------------------------

func TestUnwrap_Ok(t *testing.T) {
	v, err := result.Ok("foo").Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "foo", v)
}

func TestUnwrap_Empty(t *testing.T) {
	v, err := result.Empty[string]().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "", v) // zero value
}

func TestUnwrap_Error(t *testing.T) {
	v, err := result.Error[string](io.EOF).Unwrap()
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, "", v) // zero value, not the would-be value
}

// Empty + Ok-with-zero-value are indistinguishable by Unwrap alone.
// Empty() disambiguates them.
func TestUnwrap_EmptyVsZeroOk(t *testing.T) {
	emptyR := result.Empty[int]()
	zeroR := result.Ok(0)

	v1, err1 := emptyR.Unwrap()
	v2, err2 := zeroR.Unwrap()

	assert.Equal(t, v1, v2)
	assert.Equal(t, err1, err2)
	// Disambiguation:
	assert.True(t, emptyR.Empty())
	assert.False(t, zeroR.Empty())
}

// -----------------------------------------------------------------------------
// Convenience views
// -----------------------------------------------------------------------------

func TestOr(t *testing.T) {
	assert.Equal(t, "v", result.Ok("v").Or("def"))
	assert.Equal(t, "def", result.Empty[string]().Or("def"))
	assert.Equal(t, "def", result.Error[string](io.EOF).Or("def"))
}

func TestOrElse(t *testing.T) {
	calls := 0
	fn := func() string {
		calls++
		return "computed"
	}

	// Ok: fn must NOT be called.
	assert.Equal(t, "v", result.Ok("v").OrElse(fn))
	assert.Equal(t, 0, calls, "fn must not run when Result is Ok")

	// Empty: fn called.
	assert.Equal(t, "computed", result.Empty[string]().OrElse(fn))
	assert.Equal(t, 1, calls)

	// Error: fn called.
	assert.Equal(t, "computed", result.Error[string](io.EOF).OrElse(fn))
	assert.Equal(t, 2, calls)
}

func TestMust_Ok(t *testing.T) {
	assert.Equal(t, 7, result.Ok(7).Must())
}

func TestMust_Empty_Panics(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Must on Empty must panic")
		err, ok := r.(error)
		require.True(t, ok, "panic value must be an error")
		assert.ErrorIs(t, err, result.ErrEmpty)
	}()
	result.Empty[int]().Must()
}

func TestMust_Error_PanicsWithUnderlying(t *testing.T) {
	want := errors.New("boom")
	defer func() {
		r := recover()
		require.NotNil(t, r, "Must on Error must panic")
		err, ok := r.(error)
		require.True(t, ok, "panic value must be an error")
		assert.Same(t, want, err, "panic must carry the underlying error verbatim")
	}()
	result.Error[int](want).Must()
}

// -----------------------------------------------------------------------------
// Match — exactly one branch fires; nil handlers are no-ops
// -----------------------------------------------------------------------------

func TestMatch_Ok(t *testing.T) {
	var seen string
	result.Ok("hi").Match(
		func(v string) { seen = "ok:" + v },
		func() { seen = "empty" },
		func(e error) { seen = "err:" + e.Error() },
	)
	assert.Equal(t, "ok:hi", seen)
}

func TestMatch_Empty(t *testing.T) {
	var seen string
	result.Empty[string]().Match(
		func(v string) { seen = "ok:" + v },
		func() { seen = "empty" },
		func(e error) { seen = "err:" + e.Error() },
	)
	assert.Equal(t, "empty", seen)
}

func TestMatch_Error(t *testing.T) {
	var seen string
	result.Error[string](errors.New("boom")).Match(
		func(v string) { seen = "ok:" + v },
		func() { seen = "empty" },
		func(e error) { seen = "err:" + e.Error() },
	)
	assert.Equal(t, "err:boom", seen)
}

// nil handlers must be silently skipped, not panic.
func TestMatch_NilHandlers(t *testing.T) {
	assert.NotPanics(t, func() { result.Ok(1).Match(nil, nil, nil) })
	assert.NotPanics(t, func() { result.Empty[int]().Match(nil, nil, nil) })
	assert.NotPanics(t, func() { result.Error[int](io.EOF).Match(nil, nil, nil) })
}

// -----------------------------------------------------------------------------
// Value semantics — Result is a value type; copies are independent
// -----------------------------------------------------------------------------

func TestResult_IsValueType(t *testing.T) {
	a := result.Ok(1)
	b := a
	// Mutating one of the underlying fields cannot happen via API, but
	// the copy must still report identical state, proving the struct
	// was copied by value.
	assert.True(t, a.IsOk())
	assert.True(t, b.IsOk())

	// Independence across reassignment.
	a = result.Empty[int]()
	assert.True(t, a.IsEmpty())
	assert.True(t, b.IsOk(), "b must be unaffected by reassigning a")
}

// -----------------------------------------------------------------------------
// Result does NOT implement Go's error interface
// -----------------------------------------------------------------------------
//
// Result.Error() returns error; Go's error interface requires Error()
// string. The signatures must remain incompatible so a Result is never
// silently treated as an error by Go's type system.
func TestResult_DoesNotImplementErrorInterface(t *testing.T) {
	var r any = result.Error[int](io.EOF)
	_, ok := r.(error)
	assert.False(t, ok, "Result must not satisfy the error interface")
}
