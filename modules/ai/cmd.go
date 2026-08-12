// Package ai exposes the `cly ai` command group: visibility into which
// provider cly's AI features will use and why.
package ai

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	coreai "github.com/yurifrl/cly/pkg/ai"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

var Cmd = &cobra.Command{
	Use:   "ai",
	Short: "AI provider inspection",
}

func Register(parent *cobra.Command) {
	parent.AddCommand(Cmd)
}

func init() {
	Cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show which AI provider is selected and why",
		RunE: func(cmd *cobra.Command, args []string) error {
			return status()
		},
	})
}

func status() error {
	// Force a resolution so the decision exists even if nothing else ran.
	r := coreai.LoadConfigWith(nil)
	if err := coreai.LastSelectionError(); err != nil {
		return fmt.Errorf("ai config error: %w", err)
	}
	d := coreai.LastDecision()
	if d == nil {
		fmt.Println("ai: no providers configured; using library defaults (anthropic)")
		return nil
	}
	if r == nil {
		fmt.Println("ai: disabled")
		return nil
	}
	fmt.Printf("picked: %s (%s)\n", d.Picked, d.Reason)
	fmt.Printf("provider: %s  model: %s\n", r.Provider, r.Model)
	if r.BaseURL != "" {
		fmt.Printf("base_url: %s\n", r.BaseURL)
	}
	if r.APIKeyEnv != "" {
		fmt.Printf("api_key: $%s %s\n", r.APIKeyEnv, setUnset(os.Getenv(r.APIKeyEnv) != ""))
	} else if r.APIKey != "" {
		fmt.Println("api_key: (literal, set)")
	}
	fmt.Println()
	fmt.Println("context:")
	fmt.Printf("  user=%s host=%s arch=%s os=%s dir=%s\n",
		d.Context.User, d.Context.Host, d.Context.Arch, d.Context.OS, d.Context.Dir)
	if len(d.EnvRefs) > 0 {
		names := make([]string, 0, len(d.EnvRefs))
		for n := range d.EnvRefs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Printf("  env.%s=%s\n", n, setUnset(d.EnvRefs[n]))
		}
	}
	fmt.Println()
	fmt.Println("entries:")
	for _, e := range d.Entries {
		note := e.Note
		if note == "" {
			note = "-"
		}
		fmt.Printf("  %-20s matched=%-5v weight=%-3d %s\n", e.Name, e.Matched, e.Weight, note)
	}
	if !pkgconfig.Get().App.Debug {
		fmt.Println("\n(debug off: run with CLY_APP_DEBUG=true to log selection on AI calls)")
	}
	return nil
}

func setUnset(b bool) string {
	if b {
		return "(set)"
	}
	return "(unset)"
}
