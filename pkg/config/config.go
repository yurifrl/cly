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

bundle:
  go_file: ~/.config/Gofile
  js_file: ~/.config/Jsfile
  python_file: ~/.config/Pythonfile
  brew_file: ~/.config/Brewfile

theme:
  style: charm

notify:
  sound: false
  use_beeep: true
  use_terminal_notifier: true
  use_zellij_status: true
  use_zellij_notify: true
  icon: ""
  types:
    notification:
      title: "🔔 Claude Task"
      subtitle: "Starting"
      message: "New task"
      sound: "Glass"
      group: "claude-task-start"
      zellij_status: "🔔 Task notification"
      zellij_event: "notification"
    stop:
      title: "✅ Claude Complete"
      subtitle: "Finished"
      message: "Task completed"
      sound: "Blow"
      group: "claude-task-complete"
      zellij_status: "✅ Task completed"
      zellij_event: "stop"
    hook:
      title: "🪝 Claude Hook"
      subtitle: "Triggered"
      message: "Hook event received"
      sound: "Submarine"
      group: "claude-hook"
      zellij_status: "🪝 Hook triggered"
      zellij_event: "hook"
    default:
      title: "📢 Claude"
      subtitle: "Notification"
      message: "Claude notification received"
      sound: "Ping"
      group: "claude-default"
      zellij_status: "📢 Claude notification"
      zellij_event: "default"

modules:
  uuid:
    default_version: v4
  demo:
    show_count: true
  dotfiles:
    directory: ~/DotFiles
    zellij_plugins_dir: ~/.config/zellij/plugins
  # Example: Use 1Password secrets with op:// references
  # backup:
  #   gcs_bucket: op://Personal/gcs-backup/bucket-name
  #   gcs_token: op://Personal/gcs-backup/token
`)

type NotifyType struct {
	Title        string `yaml:"title" mapstructure:"title"`
	Subtitle     string `yaml:"subtitle" mapstructure:"subtitle"`
	Message      string `yaml:"message" mapstructure:"message"`
	Sound        string `yaml:"sound" mapstructure:"sound"`
	Group        string `yaml:"group" mapstructure:"group"`
	ZellijStatus string `yaml:"zellij_status" mapstructure:"zellij_status"`
	ZellijEvent  string `yaml:"zellij_event" mapstructure:"zellij_event"`
	Hook         string `yaml:"hook" mapstructure:"hook"`
	Matcher      string `yaml:"matcher" mapstructure:"matcher"`
}

type NotifyConfig struct {
	Sound               bool                    `yaml:"sound" mapstructure:"sound"`
	UseBeeep            bool                    `yaml:"use_beeep" mapstructure:"use_beeep"`
	UseTerminalNotifier bool                    `yaml:"use_terminal_notifier" mapstructure:"use_terminal_notifier"`
	UseZellijStatus     bool                    `yaml:"use_zellij_status" mapstructure:"use_zellij_status"`       // zjstatus plugin
	UseZellijNotify     bool                    `yaml:"use_zellij_notify" mapstructure:"use_zellij_notify"`       // notify plugin
	Icon                string                  `yaml:"icon" mapstructure:"icon"`
	Types               map[string]NotifyType   `yaml:"types" mapstructure:"types"`
}

type Config struct {
	App struct {
		Name      string `yaml:"name"`
		Debug     bool   `yaml:"debug"`
		ConfigDir string `yaml:"config_dir"`
		DataDir   string `yaml:"data_dir"`
	} `yaml:"app"`
	Bundle struct {
		GoFile     string `yaml:"go_file"`
		JsFile     string `yaml:"js_file"`
		PythonFile string `yaml:"python_file"`
		BrewFile   string `yaml:"brew_file"`
	} `yaml:"bundle"`
	Theme struct {
		Style string `yaml:"style"`
	} `yaml:"theme"`
	Notify  NotifyConfig                      `yaml:"notify" mapstructure:"notify"`
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
