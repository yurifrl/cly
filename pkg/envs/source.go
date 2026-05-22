package envs

import (
	"os"
	"sync"
)

// Source is the abstraction over the environment. Every read and write
// in pkg/envs goes through the active Source, which lets tests inject
// an in-memory implementation without touching the real process
// environment.
//
// Implementations must be safe for concurrent use; the package
// serializes Source swaps but does not synchronize individual calls
// against in-flight reads.
type Source interface {
	// Lookup returns (value, true) if name is set, (zero, false)
	// otherwise. Unlike os.LookupEnv, an empty string IS considered
	// "set" — callers in this package treat empty values as Empty,
	// not as Ok(""). Sources should report empty values verbatim
	// and let the package layer decide.
	Lookup(name string) (string, bool)

	// Set assigns value to name. Returning an error supports sources
	// that may fail (e.g., an OS that refuses certain names); the
	// default os-backed source returns nil.
	Set(name, value string) error

	// Unset removes name from the environment. No-op if absent.
	Unset(name string)
}

// -----------------------------------------------------------------------------
// Active source — package-level, swappable
// -----------------------------------------------------------------------------

var (
	sourceMu sync.RWMutex
	source   Source = osSource{}
)

// Use replaces the active Source. Returns the previous source so
// tests can restore it cleanly:
//
//	prev := envs.Use(envs.NewMapSource(...))
//	defer envs.Use(prev)
//
// Passing nil resets the source to the default os-backed implementation.
func Use(s Source) Source {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	prev := source
	if s == nil {
		s = osSource{}
	}
	source = s
	return prev
}

// active returns the current Source under read-lock.
func active() Source {
	sourceMu.RLock()
	defer sourceMu.RUnlock()
	return source
}

// -----------------------------------------------------------------------------
// osSource — default, backed by os.Getenv/Setenv/Unsetenv
// -----------------------------------------------------------------------------

// osSource is a stateless adapter over the real process environment.
// It is the default active source.
type osSource struct{}

// OSSource returns a Source backed by the real process environment.
// Equivalent to the package's default source.
func OSSource() Source { return osSource{} }

func (osSource) Lookup(name string) (string, bool) { return os.LookupEnv(name) }
func (osSource) Set(name, value string) error      { return os.Setenv(name, value) }
func (osSource) Unset(name string)                 { os.Unsetenv(name) }

// -----------------------------------------------------------------------------
// MapSource — in-memory implementation for tests
// -----------------------------------------------------------------------------

// MapSource is an in-memory Source for tests. It is safe for
// concurrent use.
type MapSource struct {
	mu sync.RWMutex
	m  map[string]string
}

// NewMapSource returns a MapSource pre-populated with seed. The seed
// map is copied; later mutations of the seed do not affect the source.
func NewMapSource(seed map[string]string) *MapSource {
	m := make(map[string]string, len(seed))
	for k, v := range seed {
		m[k] = v
	}
	return &MapSource{m: m}
}

func (s *MapSource) Lookup(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[name]
	return v, ok
}

func (s *MapSource) Set(name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[name] = value
	return nil
}

func (s *MapSource) Unset(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, name)
}
