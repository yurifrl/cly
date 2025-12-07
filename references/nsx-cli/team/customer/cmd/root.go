package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "customer",
	Short: "Customers tools for NSX",
	Long:  `This command group contains tools for managing customers in NSX.`,
}
