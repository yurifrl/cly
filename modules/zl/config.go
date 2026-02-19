package zl

import (
	"github.com/yurifrl/cly/pkg/config"
)

// Function variables for testing
var (
	getConfigFunc = getConfigReal
	setConfigFunc = config.Set
)

// ZlConfig holds zl module configuration
type ZlConfig struct {
	SessionDirs   map[string]string
	AutoZoxide    bool
	UpdateZoxide  bool
}

// LoadZlConfig loads zl configuration from global config
func LoadZlConfig() ZlConfig {
	cfg := ZlConfig{
		AutoZoxide:   true,
		UpdateZoxide: true,
		SessionDirs:  make(map[string]string),
	}

	zlData := getConfigFunc()
	if zlData == nil {
		return cfg
	}

	if autoZoxide, ok := zlData["auto_zoxide"].(bool); ok {
		cfg.AutoZoxide = autoZoxide
	}
	if updateZoxide, ok := zlData["update_zoxide"].(bool); ok {
		cfg.UpdateZoxide = updateZoxide
	}
	if sessionDirs, ok := zlData["session_dirs"].(map[string]interface{}); ok {
		for k, v := range sessionDirs {
			if str, ok := v.(string); ok {
				cfg.SessionDirs[k] = str
			}
		}
	}

	return cfg
}

// SaveSessionMapping saves a session→directory mapping to config
func SaveSessionMapping(session, dir string) error {
	zlData := getConfigFunc()
	if zlData == nil {
		zlData = make(map[string]interface{})
	}

	sessionDirs := make(map[string]string)
	if existing, ok := zlData["session_dirs"].(map[string]interface{}); ok {
		for k, v := range existing {
			if str, ok := v.(string); ok {
				sessionDirs[k] = str
			}
		}
	}

	sessionDirs[session] = dir
	return setConfigFunc("modules.zl.session_dirs", sessionDirs)
}

func getConfigReal() map[string]interface{} {
	cfg := config.Get()
	if cfg == nil {
		return nil
	}
	if zlData, ok := cfg.Modules["zl"]; ok {
		return zlData
	}
	return nil
}
