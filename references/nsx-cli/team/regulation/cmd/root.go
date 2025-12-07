package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "regulation",
	Short: "Regulation management tools for NSX",
	Long:  `This command group contains tools for managing code regulation markers and documentation.`,
}
