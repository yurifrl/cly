package config

import (
	"bytes"
	"os"
	"path/filepath"

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

modules:
  uuid:
    default_version: v4
  demo:
    show_count: true
  dotfiles:
    directory: ~/DotFiles
    zellij_plugins_dir: ~/.config/zellij/plugins
  backup:
    gcs_bucket: ""
`)

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
	Modules map[string]map[string]interface{} `yaml:"modules"`
}

var globalConfig *Config

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
