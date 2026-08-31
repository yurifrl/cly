package ompwrap

import (
	"github.com/yurifrl/cly/pkg/config"
)

// configGetString reads a config key via the shared cly config loader.
func configGetString(key string) string {
	return config.GetString(key)
}
