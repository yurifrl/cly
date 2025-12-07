package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/awssdk"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

var (
	outputFile string
	region     string
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Interactive AWS Secrets Manager selector with dynamic filtering",
	Long: `Interactive AWS Secrets Manager selector with dynamic filtering.

Controls:
  - Type to filter secrets dynamically
  - Select a secret to view/copy/save
  - Choose action: view, copy key, copy value, or save to file`,
	RunE: runSecretsCommand,
}

func init() {
	secretsCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Save selected secret to a file")
	secretsCmd.Flags().StringVarP(&region, "region", "r", "us-east-1", "AWS region")
	RootCmd.AddCommand(secretsCmd)
}

// spinnerChars are the Unicode characters used for the loading animation
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// showSpinner displays a loading animation while a background operation runs
func showSpinner(ctx context.Context, message string) {
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\r\033[K") // Clear the line
			return
		case <-ticker.C:
			fmt.Printf("\r🔍 %s %s", message, spinnerChars[i%len(spinnerChars)])
			i++
		}
	}
}

// showProgressiveSpinner displays a loading animation with a dynamic counter
func showProgressiveSpinner(ctx context.Context, message string, counter *int) {
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\r\033[K") // Clear the line
			return
		case <-ticker.C:
			if counter != nil {
				fmt.Printf("\r🔍 %s %s (found %d)", message, spinnerChars[i%len(spinnerChars)], *counter)
			} else {
				fmt.Printf("\r🔍 %s %s", message, spinnerChars[i%len(spinnerChars)])
			}
			i++
		}
	}
}

func runSecretsCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	interact.Info("🔍 Connecting to AWS Secrets Manager...")

	// Create the secrets manager client
	client, err := awssdk.NewSecretsManagerClient(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Start progressive loading with async secrets listing
	spinnerCtx, cancelSpinner := context.WithCancel(ctx)
	var wg sync.WaitGroup

	secretNames := make([]string, 0)
	var secretCount int

	wg.Add(1)
	go func() {
		defer wg.Done()
		showProgressiveSpinner(spinnerCtx, "Fetching secrets from AWS Secrets Manager...", &secretCount)
	}()

	// Start async listing
	secretChan := client.ListSecretsAsync(ctx)

	// Collect secrets as they come in
	for result := range secretChan {
		if result.Error != nil {
			cancelSpinner()
			wg.Wait()
			return fmt.Errorf("failed to list secrets: %w", result.Error)
		}

		if result.Name != "" {
			secretNames = append(secretNames, result.Name)
			secretCount = len(secretNames) // Update counter for spinner
		}
	}

	// Stop spinner
	cancelSpinner()
	wg.Wait()

	if len(secretNames) == 0 {
		interact.Info("No secrets found in AWS Secrets Manager")
		return nil
	}

	// Sort the collected secret names
	sort.Strings(secretNames)

	interact.Success("✅ Found %d secrets - Ready to search!", len(secretNames))

	// Interactive secret selection with filtering
	var selectedSecret string
	prompt := &survey.Select{
		Message:  fmt.Sprintf("Select a secret (%d available):", len(secretNames)),
		Options:  secretNames,
		PageSize: 15,
		Filter:   searchFilter,
	}

	err = survey.AskOne(prompt, &selectedSecret)
	if err != nil {
		return fmt.Errorf("secret selection cancelled: %w", err)
	}

	interact.Info("Selected secret: %s", selectedSecret)

	// Start spinner for fetching secret value
	spinnerCtx2, cancelSpinner2 := context.WithCancel(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		showSpinner(spinnerCtx2, "Retrieving secret value...")
	}()

	// Get the secret value using the existing AWS SDK function
	secretMap, err := awssdk.GetSecretMapWithRegion(selectedSecret, region)

	// Stop spinner
	cancelSpinner2()
	wg.Wait()

	if err != nil {
		return fmt.Errorf("failed to retrieve secret value: %w", err)
	}

	// Check if it's a single value or multiple keys
	if len(secretMap) == 1 && secretMap["value"] != "" {
		// Single value secret
		return handleSingleValueSecret(selectedSecret, secretMap["value"])
	} else if len(secretMap) > 1 {
		// Multi-key JSON secret
		return handleMultiKeySecret(selectedSecret, secretMap)
	} else {
		interact.Warn("Secret appears to be empty")
		return nil
	}
}

func handleSingleValueSecret(secretName, secretValue string) error {
	interact.Info("Secret content (single value):")
	fmt.Printf("%s\n\n", secretValue)

	var action string
	prompt := &survey.Select{
		Message: "What would you like to do?",
		Options: []string{
			"Copy to clipboard",
			"Save to file",
			"Nothing",
		},
	}

	err := survey.AskOne(prompt, &action)
	if err != nil {
		return err
	}

	switch action {
	case "Copy to clipboard":
		err := clipboard.WriteAll(secretValue)
		if err != nil {
			interact.Warn("Failed to copy to clipboard: %v", err)
			interact.Info("Secret value: %s", secretValue)
		} else {
			interact.Success("✅ Secret copied to clipboard")
		}

	case "Save to file":
		filename := outputFile
		if filename == "" {
			prompt := &survey.Input{
				Message: "Enter filename to save secret:",
			}
			err := survey.AskOne(prompt, &filename)
			if err != nil {
				return err
			}
		}

		if filename != "" {
			err := os.WriteFile(filename, []byte(secretValue), 0o600)
			if err != nil {
				return fmt.Errorf("failed to save secret to file: %w", err)
			}
			interact.Success("✅ Secret saved to %s", filename)
		}
	}

	return nil
}

func handleMultiKeySecret(secretName string, secretMap map[string]string) error {
	// Get sorted keys for consistent display
	keys := make([]string, 0, len(secretMap))
	for key := range secretMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	interact.Info("Secret contains the following keys:")
	for _, key := range keys {
		fmt.Printf("  • %s\n", key)
	}
	fmt.Println()

	var action string
	prompt := &survey.Select{
		Message: "What would you like to do?",
		Options: []string{
			"View all",
			"Copy specific key",
			"Copy specific value",
			"Save to file",
			"Nothing",
		},
	}

	err := survey.AskOne(prompt, &action)
	if err != nil {
		return err
	}

	switch action {
	case "View all":
		interact.Info("Secret content:")
		// Convert back to JSON for pretty printing
		jsonBytes, err := json.MarshalIndent(secretMap, "", "  ")
		if err != nil {
			// Fallback to simple key-value display
			for _, key := range keys {
				fmt.Printf("%s: %s\n", key, secretMap[key])
			}
		} else {
			fmt.Printf("%s\n", string(jsonBytes))
		}

	case "Copy specific key":
		var selectedKey string
		keyPrompt := &survey.Select{
			Message: "Select key to copy:",
			Options: keys,
		}
		err := survey.AskOne(keyPrompt, &selectedKey)
		if err != nil {
			return err
		}

		err = clipboard.WriteAll(selectedKey)
		if err != nil {
			interact.Warn("Failed to copy to clipboard: %v", err)
			interact.Info("Key: %s", selectedKey)
		} else {
			interact.Success("✅ Key '%s' copied to clipboard", selectedKey)
		}

	case "Copy specific value":
		var selectedKey string
		keyPrompt := &survey.Select{
			Message: "Select key to copy its value:",
			Options: keys,
		}
		err := survey.AskOne(keyPrompt, &selectedKey)
		if err != nil {
			return err
		}

		value := secretMap[selectedKey]
		err = clipboard.WriteAll(value)
		if err != nil {
			interact.Warn("Failed to copy to clipboard: %v", err)
			interact.Info("Value: %s", value)
		} else {
			interact.Success("✅ Value for '%s' copied to clipboard", selectedKey)
		}

	case "Save to file":
		filename := outputFile
		if filename == "" {
			prompt := &survey.Input{
				Message: "Enter filename to save secret:",
			}
			err := survey.AskOne(prompt, &filename)
			if err != nil {
				return err
			}
		}

		if filename != "" {
			// Save as pretty-printed JSON
			jsonBytes, err := json.MarshalIndent(secretMap, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal secret to JSON: %w", err)
			}

			err = os.WriteFile(filename, jsonBytes, 0o600)
			if err != nil {
				return fmt.Errorf("failed to save secret to file: %w", err)
			}
			interact.Success("✅ Secret saved to %s", filename)
		}
	}

	return nil
}

// searchFilter implements case-insensitive substring filtering
func searchFilter(filterValue string, optionValue string, optionIndex int) bool {
	return strings.Contains(
		strings.ToLower(optionValue),
		strings.ToLower(filterValue),
	)
}
