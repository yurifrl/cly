package oi

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var verbosity int

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "spell-checker [word or phrase]",
		Short: "Spell checker with explanations (pt-BR & English)",
		Long: `AI-powered spell checker for English and Brazilian Portuguese.

For single words: detects language, corrects spelling, shows definition and explanation.
For phrases: detects if you're asking about a word or want the phrase fixed.

Uses LLM for rich explanations — like having a dictionary that also roasts your spelling.

Name inspired by Homer Simpson: "I am so smart, S-M-R-T!"`,
		Example: `  cly spell-checker "laxante"
  cly spell-checker "definately"
  cly spell-checker "eu tenho um problma com isso"
  cly spell-checker -V 0 "speling"              # just the corrected word
  echo "becuase" | cly spell-checker            # pipe mode
  cly spell-checker                              # interactive mode`,
		Aliases: []string{"oi"},
		RunE:    run,
	}
	cmd.Flags().IntVarP(&verbosity, "verbosity", "V", 1, "0=corrected word only, 1=colorful+explanation (default), 2=full dictionary")
	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	// Mode 1: args
	if len(args) > 0 {
		input := strings.Join(args, " ")
		return processInput(input)
	}

	// Mode 2: pipe
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				if err := processInput(line); err != nil {
					return err
				}
			}
		}
		return scanner.Err()
	}

	// Mode 3: interactive TUI
	p := tea.NewProgram(newInteractiveModel())
	_, err := p.Run()
	return err
}

func processInput(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("empty input")
	}
	return check(input)
}
