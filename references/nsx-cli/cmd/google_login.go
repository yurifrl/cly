package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"

	"github.com/NSXBet/nsx-cli/shared/googlex"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

func init() {
	RootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Google drive API",
	Long: `Login to Google drive API to allow reading and writing to Google drive.
This will open your browser for authentication and save the credentials locally.`,
	RunE: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	if err := googlex.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     googlex.ClientID(),
		ClientSecret: googlex.ClientSecret(),
		Scopes: []string{
			sheets.DriveFileScope,
			sheets.DriveScope,
			sheets.SpreadsheetsScope,
		},
		Endpoint: google.Endpoint,
	}

	state := fmt.Sprintf("state-%d", time.Now().Unix())
	authServer := googlex.NewAuthServer(oauthConfig, state)

	if _, err := authServer.Start(); err != nil {
		return fmt.Errorf("failed to start auth server: %v", err)
	}

	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	if err := googlex.OpenBrowser(authURL); err != nil {
		interact.Error("failed to open browser: %v", err)
		interact.Info("Please open the following URL in your browser: %s", authURL)
	}

	authCode := <-authServer.AuthCode

	ctx := context.Background()

	token, err := oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		interact.Error("failed to exchange token: %v", err)
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	if err := googlex.SaveToken(token); err != nil {
		interact.Error("failed to save token: %v", err)
		return fmt.Errorf("failed to save token: %w", err)
	}

	authServer.Shutdown(ctx)
	interact.Success("Login successful")

	return nil
}
