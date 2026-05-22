package envs

import "fmt"

// ParseError is the canonical error returned in the Error state of a
// Result when an env var is set but cannot be parsed into the expected
// type. It carries enough context for callers to identify the offender
// without parsing log lines.
//
// Use errors.As to extract structured information:
//
//	var pe *envs.ParseError
//	if errors.As(err, &pe) {
//	    log.Warnf("env %s: bad %s value %q", pe.Name, pe.Kind, pe.Value)
//	}
type ParseError struct {
	// Name is the canonical env var name (the first key tried).
	Name string
	// Kind is a short type tag — "bool", "int", "duration", etc.
	Kind string
	// Value is the raw string that failed to parse.
	Value string
	// Cause is the underlying parse error (e.g. strconv.NumError).
	Cause error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("envs: %s=%q is not a valid %s: %v", e.Name, e.Value, e.Kind, e.Cause)
}

// Unwrap exposes the underlying parse error for errors.Is / errors.As
// chains.
func (e *ParseError) Unwrap() error { return e.Cause }
