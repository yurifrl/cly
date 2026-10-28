package envs

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Cmd is the envs root command.
var Cmd = &cobra.Command{
	Use:   "envs",
	Short: "Load environment variables from 1Password",
	Long: `Fetches secrets from 1Password and caches them in macOS Keychain.

  cly envs                 Fetch from 1Password → store in Keychain
  cly envs --fish          Read from Keychain → fish format (instant)
  cly envs --plain         Read from Keychain → KEY=VALUE format (instant)
  cly envs --refresh       Force re-fetch from 1Password
  cly envs --refresh --fish  Re-fetch + output fish format

If Keychain is empty, --fish/--plain will fetch automatically.
Disable Keychain with keychain: false in ~/.config/cly/config.yaml.`,
	RunE: runEnvs,
}

var (
	fishFlag     bool
	plainFlag    bool
	refreshFlag  bool
	sectionsFlag bool
	opBinaryFlag string
	vaultFlag    string
	accountFlag  string
	secretsFlag  []string
)

func init() {
	Cmd.Flags().BoolVar(&fishFlag, "fish", false, "Output fish format (set -gx)")
	Cmd.Flags().BoolVar(&plainFlag, "plain", false, "Output .env format (KEY=VALUE)")
	Cmd.Flags().BoolVar(&refreshFlag, "refresh", false, "Force re-fetch from 1Password")
	Cmd.Flags().BoolVar(&sectionsFlag, "sections", false, "Also output section-prefixed vars (GENERAL_*, WORK_*, etc.)")
	Cmd.Flags().StringVar(&opBinaryFlag, "op", "op", "Path to the 1Password CLI")
	Cmd.Flags().StringVar(&vaultFlag, "vault", DefaultVault, "1Password vault")
	Cmd.Flags().StringVar(&accountFlag, "account", DefaultAccount, "1Password account")
	Cmd.Flags().StringSliceVar(&secretsFlag, "secrets", []string{"envs-*"}, "Item name patterns (glob or explicit)")

	Cmd.AddCommand(installCmd)
}

// Register adds the envs commands to the parent command.
func Register(parent *cobra.Command) {
	parent.AddCommand(Cmd)
}

func runEnvs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	config := LoadConfig()
	if cmd.Flags().Changed("vault") {
		config.Vault = vaultFlag
	}
	if cmd.Flags().Changed("account") {
		config.Account = accountFlag
	}
	if cmd.Flags().Changed("secrets") {
		config.Secrets = secretsFlag
	}
	if !cmd.Flags().Changed("sections") && config.Sections {
		sectionsFlag = true
	}

	wantOutput := fishFlag || plainFlag
	keychainEnabled := config.Keychain

	// If keychain disabled: always fetch and output
	if !keychainEnabled {
		fields, err := fetchFromOP(ctx, config, !wantOutput)
		if err != nil {
			return err
		}
		if wantOutput {
			writeOutput(os.Stdout, fields, fishFlag, sectionsFlag)
		}
		return nil
	}

	// Keychain enabled
	if wantOutput && !refreshFlag {
		// Try reading from Keychain first
		fields, err := keychainReadAll()
		if err == nil && len(fields) > 0 {
			writeOutput(os.Stdout, fields, fishFlag, sectionsFlag)
			return nil
		}
		// Keychain empty — fall through to fetch
	}

	// Fetch from 1Password
	fields, err := fetchFromOP(ctx, config, !wantOutput)
	if err != nil {
		return err
	}

	// Store in Keychain
	if err := keychainStoreAll(fields); err != nil {
		if !wantOutput {
			fmt.Fprintf(os.Stderr, "⚠ Keychain store failed: %v\n", err)
		}
	} else if !wantOutput {
		fmt.Fprintf(os.Stderr, "✓ Cached %d variables in macOS Keychain\n", len(fields))
	}

	// Output if requested
	if wantOutput {
		writeOutput(os.Stdout, fields, fishFlag, sectionsFlag)
	}

	return nil
}

// fetchFromOP fetches all secrets from 1Password.
func fetchFromOP(ctx context.Context, config Config, verbose bool) ([]Field, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "Fetching from 1Password (account: %s, vault: %s, patterns: %s)...\n",
			config.Account, config.Vault, strings.Join(config.Secrets, ", "))
	}

	names, err := discoverSecrets(ctx, opBinaryFlag, config)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf(
			"no items found matching patterns: %v\n  account: %s\n  vault: %s\n\nCreate items in 1Password matching the pattern, or override with:\n  cly envs --vault <vault> --secrets <pattern>",
			config.Secrets,
			config.Account,
			config.Vault,
		)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d items, fetching in parallel...\n", len(names))
	}

	results := fetchAll(ctx, opBinaryFlag, config, names)
	var allFields []Field
	var errCount int
	for result := range results {
		if result.Err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "✗ %s: %v\n", result.Name, result.Err)
			}
			errCount++
			continue
		}
		filtered := filterValidFields(result.Fields)
		allFields = append(allFields, filtered...)
		if verbose {
			fmt.Fprintf(os.Stderr, "✓ %s  %d variables\n", result.Name, len(filtered))
		}
	}

	if len(allFields) == 0 {
		return nil, fmt.Errorf("no variables loaded (%d errors)", errCount)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "✓ Loaded %d variables from %d items\n", len(allFields), len(names)-errCount)
	}
	return allFields, nil
}

func writeOutput(w io.Writer, fields []Field, fish, sections bool) {
	for _, field := range fields {
		if sections {
			prefix := "GENERAL_"
			if field.Section != "" {
				prefix = strings.ToUpper(field.Section) + "_"
			}
			if fish {
				_, _ = fmt.Fprintf(w, "set -gx %s %s\n", prefix+field.Label, fishLiteral(field.Value))
			} else {
				_, _ = fmt.Fprintf(w, "%s%s=%s\n", prefix, field.Label, envQuote(field.Value))
			}
		}
		if fish {
			_, _ = fmt.Fprintf(w, "set -gx %s %s\n", field.Label, fishLiteral(field.Value))
		} else {
			_, _ = fmt.Fprintf(w, "%s=%s\n", field.Label, envQuote(field.Value))
		}
	}
}

func fishLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func envQuote(value string) string {
	if !strings.ContainsAny(value, " \t\n'\"\\$`!#&|;(){}<>?") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
