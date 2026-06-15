package aliases

import (
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/completion"
)

func Register(parent *cobra.Command) {
	// Register lazy generators — run when `cly completion fish` executes,
	// after all modules are registered on parent.
	//
	// Alias definitions (`alias p "cly pi";`) are runnable commands and must
	// load at shell startup, so they go through RegisterLazyAliases — the
	// install command writes those to fish conf.d (startup-sourced).
	//
	// Completion wrappers (`complete -c p -w 'cly pi'`) belong with the other
	// completion specs in the completions file (lazily autoloaded by fish).
	completion.RegisterLazyAliases(func() string {
		entries := GenerateAliases(parent, exec.LookPath)
		return FormatFish(entries)
	})
	completion.RegisterLazy(func() string {
		entries := GenerateAliases(parent, exec.LookPath)
		skip := completion.RegisteredAliases()
		return FormatFishCompletions(entries, skip...)
	})
}
