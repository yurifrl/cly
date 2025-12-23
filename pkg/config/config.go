package config

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// defaultConfig is the embedded default configuration
var defaultConfig = []byte(`app:
  name: cly
  debug: false
  config_dir: ~/.config/cly
  data_dir: ~/.local/share/cly
  dotfiles_dir: ~/DotFiles

theme:
  style: charm

modules:
  bundle:
    go_file: ~/.config/Gofile
    js_file: ~/.config/Jsfile
    python_file: ~/.config/Pythonfile
    brew_file: ~/.config/Brewfile
  notify:
    enabled: true
    sound: false
    use_zellij_status: true
    use_zellij_notify: true
    icon: ""
    hooks:
      notification:
        enabled: true
        title: "🔔 Claude Task"
        message: "Starting - New task [${ZELLIJ_SESSION_NAME}] ${CLAUDE_SESSION_NAME}"
        sound: "Glass"
        zellij_status: "🔔 Task notification"
        zellij_event: "notification"
      stop:
        enabled: true
        title: "✅ Claude Complete"
        message: "Finished - Task completed [${ZELLIJ_SESSION_NAME}] ${CLAUDE_SESSION_NAME}"
        sound: "Blow"
        zellij_status: "✅ Task completed"
        zellij_event: "stop"
  uuid:
    default_version: v4
  demo:
    show_count: true
  dotfiles:
    zellij_plugins_dir: ~/.config/zellij/plugins
  # Example: Use 1Password secrets with op:// references
  # backup:
  #   gcs_bucket: op://Personal/gcs-backup/bucket-name
  #   gcs_token: op://Personal/gcs-backup/token
`)

type HookConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	Title        string `yaml:"title" mapstructure:"title"`
	Message      string `yaml:"message" mapstructure:"message"`
	Sound        string `yaml:"sound" mapstructure:"sound"`
	ZellijStatus string `yaml:"zellij_status" mapstructure:"zellij_status"`
	ZellijEvent  string `yaml:"zellij_event" mapstructure:"zellij_event"`
}

type NotifyConfig struct {
	Enabled         bool                  `yaml:"enabled" mapstructure:"enabled"`
	Sound           bool                  `yaml:"sound" mapstructure:"sound"`
	UseZellijStatus bool                  `yaml:"use_zellij_status" mapstructure:"use_zellij_status"` // zjstatus plugin
	UseZellijNotify bool                  `yaml:"use_zellij_notify" mapstructure:"use_zellij_notify"` // notify plugin
	Icon            string                `yaml:"icon" mapstructure:"icon"`
	Hooks           map[string]HookConfig `yaml:"hooks" mapstructure:"hooks"`
}

type Config struct {
	App struct {
		Name        string `yaml:"name"`
		Debug       bool   `yaml:"debug"`
		ConfigDir   string `yaml:"config_dir"`
		DataDir     string `yaml:"data_dir"`
		DotFilesDir string `yaml:"dotfiles_dir"`
	} `yaml:"app"`
	Theme struct {
		Style string `yaml:"style"`
	} `yaml:"theme"`
	Modules map[string]map[string]interface{} `yaml:"modules"`
}

var globalConfig *Config

// newOpResolverFunc is a variable to allow test override
var newOpResolverFunc = func() *OpResolver {
	return NewOpResolver()
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config/cly")

	// Enable env var support with CLY_ prefix
	v.SetEnvPrefix("CLY")
	v.AutomaticEnv()

	// Try config.local.yaml first (not committed), then config.yaml
	configFound := false
	for _, configName := range []string{"config.local", "config"} {
		v.SetConfigName(configName)
		v.AddConfigPath(configDir)
		v.AddConfigPath("modules/config")
		v.AddConfigPath(".")

		if err := v.ReadInConfig(); err == nil {
			configFound = true
			break
		}
	}

	// No config file found, load embedded defaults
	if !configFound {
		if err := v.ReadConfig(bytes.NewBuffer(defaultConfig)); err != nil {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Resolve secrets in modules section
	if len(cfg.Modules) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resolver := newOpResolverFunc()
		if err := resolveSecretsInPlace(ctx, resolver, cfg.Modules); err != nil {
			return nil, err
		}
	}

	globalConfig = &cfg
	return &cfg, nil
}

func Get() *Config {
	if globalConfig == nil {
		globalConfig, _ = Load()
	}
	return globalConfig
}

// GetNotify returns the notify configuration from modules
func (c *Config) GetNotify() NotifyConfig {
	var notify NotifyConfig
	if notifyData, ok := c.Modules["notify"]; ok {
		// Convert map to NotifyConfig struct
		if enabled, ok := notifyData["enabled"].(bool); ok {
			notify.Enabled = enabled
		}
		if sound, ok := notifyData["sound"].(bool); ok {
			notify.Sound = sound
		}
		if useZellijStatus, ok := notifyData["use_zellij_status"].(bool); ok {
			notify.UseZellijStatus = useZellijStatus
		}
		if useZellijNotify, ok := notifyData["use_zellij_notify"].(bool); ok {
			notify.UseZellijNotify = useZellijNotify
		}
		if icon, ok := notifyData["icon"].(string); ok {
			notify.Icon = icon
		}
		if hooks, ok := notifyData["hooks"].(map[string]interface{}); ok {
			notify.Hooks = make(map[string]HookConfig)
			for hookName, hookData := range hooks {
				if hookMap, ok := hookData.(map[string]interface{}); ok {
					hook := HookConfig{}
					if enabled, ok := hookMap["enabled"].(bool); ok {
						hook.Enabled = enabled
					}
					if title, ok := hookMap["title"].(string); ok {
						hook.Title = os.ExpandEnv(title)
					}
					if message, ok := hookMap["message"].(string); ok {
						hook.Message = os.ExpandEnv(message)
					}
					if sound, ok := hookMap["sound"].(string); ok {
						hook.Sound = os.ExpandEnv(sound)
					}
					if zellijStatus, ok := hookMap["zellij_status"].(string); ok {
						hook.ZellijStatus = os.ExpandEnv(zellijStatus)
					}
					if zellijEvent, ok := hookMap["zellij_event"].(string); ok {
						hook.ZellijEvent = os.ExpandEnv(zellijEvent)
					}
					notify.Hooks[hookName] = hook
				}
			}
		}
	}
	return notify
}

func GetString(key string) string {
	v := viper.New()
	v.SetConfigType("yaml")

	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config/cly")

	v.SetEnvPrefix("CLY")
	v.AutomaticEnv()

	// Try config.local.yaml first, then config.yaml
	configFound := false
	for _, configName := range []string{"config.local", "config"} {
		v.SetConfigName(configName)
		v.AddConfigPath(configDir)
		v.AddConfigPath("modules/config")
		v.AddConfigPath(".")

		if err := v.ReadInConfig(); err == nil {
			configFound = true
			break
		}
	}

	// Fall back to defaults if no config found
	if !configFound {
		v.ReadConfig(bytes.NewBuffer(defaultConfig))
	}

	return v.GetString(key)
}

func GetBool(key string) bool {
	v := viper.New()
	v.SetConfigType("yaml")

	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config/cly")

	v.SetEnvPrefix("CLY")
	v.AutomaticEnv()

	// Try config.local.yaml first, then config.yaml
	configFound := false
	for _, configName := range []string{"config.local", "config"} {
		v.SetConfigName(configName)
		v.AddConfigPath(configDir)
		v.AddConfigPath("modules/config")
		v.AddConfigPath(".")

		if err := v.ReadInConfig(); err == nil {
			configFound = true
			break
		}
	}

	// Fall back to defaults if no config found
	if !configFound {
		v.ReadConfig(bytes.NewBuffer(defaultConfig))
	}

	return v.GetBool(key)
}

func Set(key string, value interface{}) error {
	homeDir, _ := os.UserHomeDir()

	// Check for env var override, otherwise use default
	configDir := os.Getenv("CLY_APP_CONFIG_DIR")
	if configDir == "" {
		configDir = filepath.Join(homeDir, ".config/cly")
	}
	configPath := filepath.Join(configDir, "config.yaml")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read existing config or start with defaults
	if err := v.ReadInConfig(); err != nil {
		v.ReadConfig(bytes.NewBuffer(defaultConfig))
	}

	v.Set(key, value)

	if err := v.WriteConfig(); err != nil {
		if err := v.SafeWriteConfig(); err != nil {
			return err
		}
	}

	return nil
}
