package completion

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// AliasCompletion holds custom fish completions for an aliased command.
type AliasCompletion struct {
	Alias string
	Lines string
}

var extraCompletions []AliasCompletion
var lazyGenerators []func() string

// RegisterAlias registers extra fish completion lines for a command alias.
// Use this for modules with custom completions (e.g., zl, mcp).
func RegisterAlias(alias string, lines string) {
	extraCompletions = append(extraCompletions, AliasCompletion{Alias: alias, Lines: lines})
}

// RegisterLazy registers a function that generates completion lines at build time.
// Use this for completions that depend on all modules being registered first.
func RegisterLazy(fn func() string) {
	lazyGenerators = append(lazyGenerators, fn)
}

// RegisteredAliases returns the names of aliases with custom completions.
func RegisteredAliases() []string {
	names := make([]string, len(extraCompletions))
	for i, ac := range extraCompletions {
		names[i] = ac.Alias
	}
	return names
}

// BuildExtraCompletions returns the combined extra completion lines.
func BuildExtraCompletions() string {
	if len(extraCompletions) == 0 && len(lazyGenerators) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n# Alias completions\n")
	for _, ac := range extraCompletions {
		b.WriteString(ac.Lines)
		if !strings.HasSuffix(ac.Lines, "\n") {
			b.WriteString("\n")
		}
	}
	for _, fn := range lazyGenerators {
		b.WriteString(fn())
	}
	return b.String()
}

func Register(parent *cobra.Command) {
	parent.CompletionOptions.DisableDefaultCmd = true

	fishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return genFish(parent, cmd.OutOrStdout())
		},
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install completions to ~/.config/fish/completions/cly.fish",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				dir = filepath.Join(home, ".config", "fish", "completions")
			}
			return installFish(parent, dir)
		},
	}
	installCmd.Flags().String("dir", "", "Override completions directory")

	fishCmd.AddCommand(installCmd)

	cmd := &cobra.Command{
		Use:       "completion [fish|bash|zsh]",
		Short:     "Generate shell completions",
		ValidArgs: []string{"fish", "bash", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("specify shell: fish, bash, or zsh")
			}
			w := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return parent.GenBashCompletion(w)
			case "zsh":
				return parent.GenZshCompletion(w)
			default:
				return fmt.Errorf("unsupported shell: %s (use 'cly completion fish' instead)", args[0])
			}
		},
	}

	cmd.AddCommand(fishCmd)
	parent.AddCommand(cmd)
}

func genFish(root *cobra.Command, w interface{ Write([]byte) (int, error) }) error {
	var buf bytes.Buffer
	if err := root.GenFishCompletion(&buf, true); err != nil {
		return err
	}

	buf.WriteString(BuildExtraCompletions())

	_, err := w.Write(buf.Bytes())
	return err
}

func installFish(root *cobra.Command, dir string) error {
	var buf bytes.Buffer
	if err := genFish(root, &buf); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	outPath := filepath.Join(dir, "cly.fish")
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Installed completions to %s\n", outPath)
	return nil
}
