package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI tools for NSX",
	Long:  `This command group contains tools for managing AI in NSX.`,
}
