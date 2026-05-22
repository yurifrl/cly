package envs

import (
	"strconv"

	"github.com/yurifrl/cly/pkg/result"
)

// This file contains the private read/write/clear/has primitives used
// by every per-var public accessor in the package. They are the ONLY
// helpers that touch raw env var name strings — no public function
// outside this package should ever pass a string env var name into
// pkg/envs APIs.
//
// Convention: each helper takes the canonical name first, followed by
// optional fallbacks (legacy aliases) tried in order.

// readString returns the first non-empty value found across name and
// fallbacks. An env var that is set to "" is treated as Empty, not as
// Ok(""), because "explicitly empty" is rarely what callers want from
// optional configuration. If you need to distinguish "set to empty"
// from "unset", use the Source directly.
func readString(name string, fallbacks ...string) result.Result[string] {
	src := active()
	for _, n := range append([]string{name}, fallbacks...) {
		if v, ok := src.Lookup(n); ok && v != "" {
			return result.Ok(v)
		}
	}
	return result.Empty[string]()
}

// readBool reads the first set value across name and fallbacks and
// parses it as a bool via strconv.ParseBool ("1", "t", "T", "TRUE",
// "true", "True", "0", "f", "F", "FALSE", "false", "False"). An unset
// var is Empty; an invalid value is Error wrapping a *ParseError.
func readBool(name string, fallbacks ...string) result.Result[bool] {
	raw := readString(name, fallbacks...)
	if !raw.IsOk() {
		// Empty -> Empty[bool]; (Error from readString never happens
		// today, but propagate generically if it ever does)
		if raw.IsError() {
			return result.Error[bool](raw.Error())
		}
		return result.Empty[bool]()
	}
	v, _ := raw.Unwrap()
	b, err := strconv.ParseBool(v)
	if err != nil {
		return result.Error[bool](&ParseError{
			Name: name, Kind: "bool", Value: v, Cause: err,
		})
	}
	return result.Ok(b)
}

// write assigns value to every name (canonical + aliases). Writing to
// aliases keeps backward-compatible readers working when the canonical
// name is migrated. The first error short-circuits.
func write(value string, names ...string) error {
	src := active()
	for _, n := range names {
		if err := src.Set(n, value); err != nil {
			return err
		}
	}
	return nil
}

// clear removes every name (canonical + aliases) from the source.
func clear(names ...string) {
	src := active()
	for _, n := range names {
		src.Unset(n)
	}
}

// has reports whether any of the given names is set (to any value,
// including empty string).
func has(names ...string) bool {
	src := active()
	for _, n := range names {
		if _, ok := src.Lookup(n); ok {
			return true
		}
	}
	return false
}
