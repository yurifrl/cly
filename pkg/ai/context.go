package ai

import (
	"os"
	"os/user"
	"runtime"
	"strings"
)

// Context is the runtime environment conditions evaluate against.
type Context struct {
	User, Host, Arch, OS, Dir string
}

// lookup resolves a condition field to its value. env.NAME reads the
// process environment. bool reports whether the field name is known.
func (c *Context) lookup(field string) (string, bool) {
	switch field {
	case "user":
		return c.User, true
	case "host":
		return c.Host, true
	case "arch":
		return c.Arch, true
	case "os":
		return c.OS, true
	case "dir":
		return c.Dir, true
	}
	if strings.HasPrefix(field, "env.") && len(field) > 4 {
		return os.Getenv(field[4:]), true
	}
	return "", false
}

// buildContext captures the current process environment for selection.
func buildContext() *Context {
	c := &Context{Arch: runtime.GOARCH, OS: runtime.GOOS}
	if u, err := user.Current(); err == nil {
		c.User = u.Username
	}
	if h, err := os.Hostname(); err == nil {
		c.Host = h
	}
	if d, err := os.Getwd(); err == nil {
		c.Dir = d
	}
	return c
}
