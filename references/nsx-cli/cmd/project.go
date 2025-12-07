package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/blueprint/builder"
)

// projectCmd adds the `nsx project` sub-command.
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Tools for managing NSX projects",
}

var (
	projectName string
	team        string
)

// projectBuildCmd adds the `nsx project build` sub-command.
var projectBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Scaffold a new service from the NSX project builder",
	RunE: func(cmd *cobra.Command, args []string) error {
		return builder.Run(builder.ProjectArguments{
			ProjectName: projectName,
			Team:        team,
			Debug:       debug,
			ProxyURL:    getProxyURL(),
		})
	},
}

// projectUpdateGolangCICmd adds the `nsx project update-golang-ci` sub-command.
var projectUpdateGolangCICmd = &cobra.Command{
	Use:   "update-golang-ci",
	Short: "Update .golangci.yml configuration from NSX gist",
	Long:  "Fetches the latest golangci-lint configuration from NSX gist and updates/creates .golangci.yml in the current directory. Only works in git repositories.",
	RunE: func(cmd *cobra.Command, args []string) error {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		return builder.FetchGolangCIConfig(getProxyURL(), currentDir, debug)
	},
}

func init() {
	RootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectBuildCmd)
	projectCmd.AddCommand(projectUpdateGolangCICmd)

	projectBuildCmd.Flags().StringVarP(&projectName, "name", "n", "", "The name of the project")
	projectBuildCmd.Flags().StringVarP(&team, "team", "t", "", "The team responsible for the project")
}
