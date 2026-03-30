package isbnform

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "isbn-form",
		Short: "ISBN-13 validation form demo",
		RunE:  run,
	}
	parent.AddCommand(cmd)
}
