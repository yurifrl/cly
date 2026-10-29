package envs

import (
	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

const (
	// DefaultAccount is the 1Password account used when unset in config.
	DefaultAccount = "my.1password.com"
	// DefaultVault is the 1Password vault used when unset in config.
	DefaultVault = "Private"
	// DefaultSections is the section-prefixed output used when unset.
	DefaultSections = true
)

// Config holds the envs loader configuration.
type Config struct {
	Account  string
	Vault    string
	Sections bool
	Cache    bool
	Secrets  []string
}

// LoadConfig reads the envs section from cly's config
// (~/.config/cly/config.local.yaml or config.yaml, keys under "envs.").
// Defaults are returned for missing values, matching the configuration the
// nsx implementation ran with on this machine.
func LoadConfig() Config {
	cfg := Config{
		Account:  pkgconfig.GetString("envs.account"),
		Vault:    pkgconfig.GetString("envs.vault"),
		Sections: pkgconfig.GetBoolWithDefault("envs.sections", DefaultSections),
		Cache:    pkgconfig.GetBoolWithDefault("envs.cache", true),
		Secrets:  pkgconfig.GetStringSlice("envs.secrets"),
	}
	if cfg.Account == "" {
		cfg.Account = DefaultAccount
	}
	if cfg.Vault == "" {
		cfg.Vault = DefaultVault
	}
	if len(cfg.Secrets) == 0 {
		cfg.Secrets = []string{"envs-*"}
	}
	return cfg
}
