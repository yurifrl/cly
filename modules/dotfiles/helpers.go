package dotfiles

import (
	"fmt"
	"strconv"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/mut"
)

// runScript executes a shell script (already on disk, +x) in the given dir.
// Used by @install and @cache execution paths.
func runScript(scriptPath, baseDir string) error {
	return mut.ExecDir(baseDir, scriptPath)
}

// dotfilesModuleValue walks `modules.dotfiles.<path...>` from pkg/config.
func dotfilesModuleValue(path ...string) interface{} {
	cfg := pkgconfig.Get()
	if cfg == nil {
		return nil
	}
	current, ok := cfg.Modules["dotfiles"]
	if !ok {
		return nil
	}
	var value interface{} = current
	for _, part := range path {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		value, ok = m[part]
		if !ok {
			return nil
		}
	}
	return value
}

func dotfilesModuleString(def string, path ...string) string {
	value := dotfilesModuleValue(path...)
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return def
	}
}
