package googlex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/NSXBet/nsx-cli/shared/interact"
)

// GetSheetsService returns an authenticated Sheets service using the cached token
func GetSheetsService(ctx context.Context) (*sheets.Service, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %v", err)
	}

	// Create OAuth config from our config.toml values
	config := &oauth2.Config{
		ClientID:     ClientID(),
		ClientSecret: ClientSecret(),
		Endpoint:     google.Endpoint,
		Scopes: []string{
			sheets.SpreadsheetsReadonlyScope,
		},
	}

	client := config.Client(ctx, token)
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create sheets service: %v", err)
	}

	return srv, nil
}

func loadToken() (*oauth2.Token, error) {
	f, err := os.Open(TokenPath())
	if err != nil {
		return nil, fmt.Errorf("token not found, please run 'phone-validate login' first: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			interact.Error("Failed to close file: %v", err)
		}
	}()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %v", err)
	}

	return token, nil
}
