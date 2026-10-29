package envs

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// cacheStoreAll writes fields to the active cache backend.
func cacheStoreAll(fields []Field) error {
	switch backend := cacheBackend(); backend {
	case "keychain":
		return keychainStoreAll(fields)
	case "age":
		return defaultAgeStore().Store(fields)
	default:
		return fmt.Errorf("unknown cache backend %q (use keychain or age)", backend)
	}
}

// cacheReadAll reads fields from the active cache backend.
func cacheReadAll() ([]Field, error) {
	switch backend := cacheBackend(); backend {
	case "keychain":
		return keychainReadAll()
	case "age":
		return defaultAgeStore().Read()
	default:
		return nil, fmt.Errorf("unknown cache backend %q (use keychain or age)", backend)
	}
}

// cacheBackend resolves the cache backend: an explicit envs.store config
// value ("keychain" or "age"; "auto" or empty = by OS), or the OS default
// (macOS = Keychain, everything else = age-encrypted file).
func cacheBackend() string {
	backend := pkgconfig.GetString("envs.store")
	if backend != "" && backend != "auto" {
		return backend
	}
	if runtime.GOOS == "darwin" {
		return "keychain"
	}
	return "age"
}

// cacheLabel returns the human-readable backend name for status messages.
func cacheLabel() string {
	if cacheBackend() != "age" {
		return "macOS Keychain"
	}
	home, _ := os.UserHomeDir()
	path := defaultAgeStore().blobPath
	if strings.HasPrefix(path, home+"/") {
		return "age cache at ~/" + path[len(home)+1:]
	}
	return "age cache at " + path
}
