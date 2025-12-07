package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/team/customer/customercfg"
)

var recoverAccountCmd = &cobra.Command{
	Use:   "recover-account",
	Short: "Recover a self-deleted customer account",
	RunE:  runRecoverAccount,
}

func init() {
	RootCmd.AddCommand(recoverAccountCmd)
}

func runRecoverAccount(cmd *cobra.Command, args []string) error {
	customerID := args[0]
	cfg, err := customercfg.GetCustomerServiceConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	interact.Info("Attempting to recover account for customer ID: %s", customerID)

	url := fmt.Sprintf(
		"%s/admin/customers/%s/self-deleted/recover",
		strings.TrimRight(cfg[customercfg.HostKey], "/"),
		customerID,
	)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Api-Token", cfg[customercfg.TokenKey])

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	interact.Success("Account %s recovered successfully", customerID)
	return nil
}
