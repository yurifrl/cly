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
    go_file: ~/.config/cly/bundles/Gofile
    js_file: ~/.config/cly/bundles/packages.json
    python_file: ~/.config/cly/bundles/Pythonfile
    brew_file: ~/.config/cly/bundles/Brewfile
    rust_file: ~/.config/cly/bundles/Rustfile
  helpy:
    preprompt: "You help answer questions based on the HELP.md below. Explain commands, shortcuts, and workflows. ONLY modify files if explicitly instructed."
    docs_dir: ~/DotFiles/docs
    ai:
      enabled: true
      provider: "anthropic"
      model: "claude-sonnet-4-5-20250929"
      api_key: ""
      api_key_env: "ANTHROPIC_API_KEY"
      system_prompt: "You are a helpful assistant. Answer questions about the document provided. Be concise and reference specific sections when possible."
  notify:
    enabled: true
    sound: false
    use_zellij_status: true
    use_zellij_notify: false
    use_zellij_attention: true
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
  memwatch:
    enabled: true
    interval: 30s
    threshold_percent: 20
    critical_percent: 10
    cooldown: 5m
    alert_on_pressure:
      - warn
      - critical
    title: "🧠 Low Memory"
    message: "Free: ${FREE}%% — Pressure: ${PRESSURE}"
    use_cmux: true
    use_desktop: true
    use_zellij: false
    sound: Basso
    top_n: 5
    process_threshold_mb: 1500
    process_growth_mb: 500
    include_top_in_alert: true
  install:
    source_dir: /Users/yuri/Workdir/Yuri/cly
  demo:
    show_count: true
  dotfiles:
    zellij_plugins_dir: ~/.config/zellij/plugins
    op:
      account: ""
    jobs:
      retries:
        enabled: true
        max_attempts: 5
        initial_delay: 2s
        multiplier: 2
        max_delay: 1m
        jitter: true
        reset_after: 10m
  statusline:
    format: "$context │ $model │ $cost │ $custom"
    context:
      enabled: false
    model:
      enabled: false
    cost:
      enabled: false
    custom:
      enabled: false
      command: ""
      timeout: 500
  # Example: Use 1Password secrets with op:// references
  # backup:
  #   gcs_bucket: op://Personal/gcs-backup/bucket-name
  #   gcs_token: op://Personal/gcs-backup/token
  git-commits:
    batch_size: 40000
    timeout: 30000
    split_prompt: ""
    ai:
      provider: "anthropic"
      model: "claude-sonnet-4-5-20250929"
      api_key: ""
      api_key_env: "ANTHROPIC_API_KEY"
`)

type HookConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	Title        string `yaml:"title" mapstructure:"title"`
	Message      string `yaml:"message" mapstructure:"message"`
	Sound        string `yaml:"sound" mapstructure:"sound"`
	ZellijStatus string `yaml:"zellij_status" mapstructure:"zellij_status"`
	ZellijEvent  string `yaml:"zellij_event" mapstructure:"zellij_event"`
}

type StatuslineConfig struct {
	Format  string                 `yaml:"format" mapstructure:"format"`
	Context StatuslineItemConfig   `yaml:"context" mapstructure:"context"`
	Model   StatuslineItemConfig   `yaml:"model" mapstructure:"model"`
	Cost    StatuslineItemConfig   `yaml:"cost" mapstructure:"cost"`
	Custom  StatuslineCustomConfig `yaml:"custom" mapstructure:"custom"`
}

type StatuslineItemConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type StatuslineCustomConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Command string `yaml:"command" mapstructure:"command"`
	Timeout int    `yaml:"timeout" mapstructure:"timeout"`
}

type NotifyConfig struct {
	Enabled            bool                  `yaml:"enabled" mapstructure:"enabled"`
	Sound              bool                  `yaml:"sound" mapstructure:"sound"`
	UseZellijStatus    bool                  `yaml:"use_zellij_status" mapstructure:"use_zellij_status"`       // zjstatus plugin
	UseZellijNotify    bool                  `yaml:"use_zellij_notify" mapstructure:"use_zellij_notify"`       // old notify plugin
	UseZellijAttention bool                  `yaml:"use_zellij_attention" mapstructure:"use_zellij_attention"` // zellij-attention plugin
	Icon               string                `yaml:"icon" mapstructure:"icon"`
	Hooks              map[string]HookConfig `yaml:"hooks" mapstructure:"hooks"`
}

type Config struct {
	App struct {
		Name        string `yaml:"name" mapstructure:"name"`
		Debug       bool   `yaml:"debug" mapstructure:"debug"`
		ConfigDir   string `yaml:"config_dir" mapstructure:"config_dir"`
		DataDir     string `yaml:"data_dir" mapstructure:"data_dir"`
		DotFilesDir string `yaml:"dotfiles_dir" mapstructure:"dotfiles_dir"`
	} `yaml:"app" mapstructure:"app"`
	Theme struct {
		Style string `yaml:"style" mapstructure:"style"`
	} `yaml:"theme" mapstructure:"theme"`
	// AI is top-level core config consumed by `pkg/ai`. Modules that use AI
	// reference these defaults; their own per-module overrides live under
	// `modules.<name>.ai`. AI is intentionally NOT a module — it's shared
	// infrastructure, on the same level as App and Theme.
	AI      map[string]interface{}            `yaml:"ai" mapstructure:"ai"`
	Modules map[string]map[string]interface{} `yaml:"modules" mapstructure:"modules"`
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

	// Try user config first
	userConfig := filepath.Join(configDir, "config.yaml")
	userLocalConfig := filepath.Join(configDir, "config.local.yaml")

	configFound := false
	for _, path := range []string{userLocalConfig, userConfig} {
		if _, err := os.Stat(path); err == nil {
			v.SetConfigFile(path)
			if err := v.ReadInConfig(); err == nil {
				configFound = true
				break
			}
		}
	}

	// Fall back to embedded defaults
	if !configFound {
		if err := v.ReadConfig(bytes.NewBuffer(defaultConfig)); err != nil {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Resolve secrets in modules section (only if op:// references exist)
	if len(cfg.Modules) > 0 && hasOpReferences(cfg.Modules) {
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
		if useZellijAttention, ok := notifyData["use_zellij_attention"].(bool); ok {
			notify.UseZellijAttention = useZellijAttention
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

// GetStatusline returns the statusline configuration from modules
func (c *Config) GetStatusline() StatuslineConfig {
	cfg := StatuslineConfig{
		Format: "$context │ $model │ $cost │ $custom",
		Custom: StatuslineCustomConfig{Timeout: 500},
	}
	if statuslineData, ok := c.Modules["statusline"]; ok {
		if format, ok := statuslineData["format"].(string); ok {
			cfg.Format = format
		}
		if contextData, ok := statuslineData["context"].(map[string]interface{}); ok {
			if enabled, ok := contextData["enabled"].(bool); ok {
				cfg.Context.Enabled = enabled
			}
		}
		if modelData, ok := statuslineData["model"].(map[string]interface{}); ok {
			if enabled, ok := modelData["enabled"].(bool); ok {
				cfg.Model.Enabled = enabled
			}
		}
		if costData, ok := statuslineData["cost"].(map[string]interface{}); ok {
			if enabled, ok := costData["enabled"].(bool); ok {
				cfg.Cost.Enabled = enabled
			}
		}
		if customData, ok := statuslineData["custom"].(map[string]interface{}); ok {
			if enabled, ok := customData["enabled"].(bool); ok {
				cfg.Custom.Enabled = enabled
			}
			if command, ok := customData["command"].(string); ok {
				cfg.Custom.Command = command
			}
			if timeout, ok := customData["timeout"].(int); ok {
				cfg.Custom.Timeout = timeout
			}
		}
	}
	return cfg
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
