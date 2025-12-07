package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/shared/skin"
	aiTeam "github.com/NSXBet/nsx-cli/team/ai/cmd"
	customerTeam "github.com/NSXBet/nsx-cli/team/customer/cmd"
	regulationTeam "github.com/NSXBet/nsx-cli/team/regulation/cmd"
	sreTeam "github.com/NSXBet/nsx-cli/team/sre/cmd"
)

var (
	skinStr string
	skinVal skin.Skin
	debug   bool
	version string
	RootCmd = &cobra.Command{
		Use:   "nsx",
		Short: "A CLI tool for NSX",
		Long:  `A CLI tool-box for NSX teams to create, update, and delete NSX resources.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			skinVal, err = skin.ParseSkin(skinStr)
			if err != nil {
				return err
			}
			skin.SetSkin(skinVal)
			if debug {
				interact.EnableDebug()
				interact.Debug("DEBUG enabled,skin: %s", skinVal)
			}
			return nil
		},
	}
)

func Execute(v string) {
	version = v
	RootCmd.AddCommand(aiTeam.RootCmd)
	RootCmd.AddCommand(customerTeam.RootCmd)
	RootCmd.AddCommand(regulationTeam.RootCmd)
	RootCmd.AddCommand(sreTeam.RootCmd)

	if err := RootCmd.Execute(); err != nil {
		interact.Error("failed to execute command: %v", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().
		StringVarP(&skinStr, "skin", "s", "betdev", "skin to use")
	RootCmd.PersistentFlags().
		BoolVarP(&debug, "debug", "d", false, "enable debug mode")
}
