package srecfg

import (
	"github.com/NSXBet/nsx-cli/shared/config"
	"github.com/NSXBet/nsx-cli/shared/skin"
)

const (
	Registry       = "op://vault/item/nsx_sre_config"
	ConfigFilename = "nsx_sre_config.toml"
)

func Load() (*Config, error) {
	return config.Load[Config](ConfigFilename)
}

type Config struct {
	Betdev      SkinConfig `toml:"betdev"`
	BetNacional SkinConfig `toml:"betnacional"`
	MRJack      SkinConfig `toml:"mrjack"`
	Mundialbet  SkinConfig `toml:"mundialbet"`
}

type SkinConfig struct {
	DBConfig map[string]DatabaseConnection `toml:"databases"`
}

type DatabaseConnection struct {
	DBInstanceIdentifier string `toml:"db_instance_identifier"`
	DBName               string `toml:"database"`
	Region               string `toml:"region"`
}

func (c *Config) ConfigSkin() SkinConfig {
	switch skin.GetSkin() {
	case skin.Betdev:
		return c.Betdev
	case skin.Betnacional:
		return c.BetNacional
	case skin.Mrjackbets:
		return c.MRJack
	case skin.Mundialbet:
		return c.Mundialbet
	default:
		return c.Betdev
	}
}
