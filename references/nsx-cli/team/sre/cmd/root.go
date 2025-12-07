package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "sre",
	Short: "SRE tools for NSX",
	Long:  `This command group contains tools for SRE team in NSX.`,
}
