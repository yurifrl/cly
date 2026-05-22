// Package envs is the single source of truth for every environment
// variable the project reads or writes.
//
// # Design
//
// Each env var is exposed as a typed function — never as an exported
// constant or generic accessor. Callers say:
//
//	envs.SessionName().Or("anonymous")
//
// not:
//
//	os.Getenv("CLY_SESSION_NAME")
//
// The literal env var name lives in exactly one place: the
// implementation of the function that reads it. Aliases (legacy names)
// are handled inside the same function. Callers never see the strings.
//
// # Return shape
//
// Read accessors return result.Result[T] (see pkg/result) so callers
// can distinguish three states:
//
//	Ok    — variable is set and parses correctly
//	Empty — variable (and aliases) absent in the source
//	Error — variable is set but invalid (e.g. unparseable bool)
//
// Write accessors (Set*, Unset*) return plain error because writes are
// imperative actions, not lookups.
//
// # Testability
//
// All reads and writes go through the package-level Source. Tests swap
// it out via Use:
//
//	envs.Use(envs.NewMapSource(map[string]string{"CLY_SESSION_NAME": "test"}))
//	defer envs.Use(envs.OSSource())
//
// # Adding a new env var
//
//   1. Pick the file matching the concern (session, cmux, zellij, …).
//   2. Add a typed accessor function and (optionally) Set/Has/Unset.
//   3. Use the private read*/write/clear/has helpers in internal.go.
//   4. Register no metadata, no map, no key constant. The function IS
//      the registration.
package envs
