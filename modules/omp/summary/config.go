package ompsummary

import (
	"strconv"
	"time"

	"github.com/yurifrl/cly/pkg/config"
)

// Config for the summary engine, read from modules.omp.summary in cly's
// config. All fields optional; defaults below.
type Config struct {
	Workers  int           // concurrent summarizer goroutines
	Interval time.Duration // discovery tick
	Debounce time.Duration // min gap before re-summarizing an errored session
	MaxAge   time.Duration // hide sessions idle longer than this
	MaxTail  int           // transcript tail bytes fed to extraction
}

func DefaultConfig() Config {
	return Config{
		Workers:  2,
		Interval: 2 * time.Second,
		Debounce: 45 * time.Second,
		MaxAge:   24 * time.Hour,
		MaxTail:  256 * 1024,
	}
}

// LoadConfig reads modules.omp.summary with type-tolerant scalars (viper may
// hand us float64, int64, int, or string depending on source format).
func LoadConfig() Config {
	c := DefaultConfig()
	mod, ok := config.Get().Modules["omp"]
	if !ok {
		return c
	}
	s, ok := mod["summary"].(map[string]interface{})
	if !ok {
		return c
	}
	if v := asInt(s["workers"]); v > 0 {
		c.Workers = v
	}
	if v := asDuration(s["interval"]); v > 0 {
		c.Interval = v
	}
	if v := asDuration(s["debounce"]); v > 0 {
		c.Debounce = v
	}
	if v := asDuration(s["max_age"]); v > 0 {
		c.MaxAge = v
	}
	if v := asInt(s["max_tail"]); v > 0 {
		c.MaxTail = v
	}
	return c
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return 0
}

func asDuration(v interface{}) time.Duration {
	switch d := v.(type) {
	case string:
		if dur, err := time.ParseDuration(d); err == nil {
			return dur
		}
		if n := asInt(d); n > 0 {
			return time.Duration(n) * time.Second
		}
	case int:
		return time.Duration(d) * time.Second
	case float64:
		return time.Duration(d) * time.Second
	}
	return 0
}
