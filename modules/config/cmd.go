package config

import (
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  "Manage CLY configuration files and settings",
}

func init() {
	ConfigCmd.AddCommand(initCmd())
	ConfigCmd.AddCommand(showCmd())
	ConfigCmd.AddCommand(getCmd())
	ConfigCmd.AddCommand(setCmd())
}

func Register(parent *cobra.Command) {
	parent.AddCommand(ConfigCmd)
}
