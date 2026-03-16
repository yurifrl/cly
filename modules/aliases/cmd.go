package aliases

import (
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/completion"
)

func Register(parent *cobra.Command) {
	// Register lazy generator — runs when `cly completion fish` executes,
	// after all modules are registered on parent.
	// Emits both alias definitions and completion wrappers so a single
	// cached file handles everything.
	completion.RegisterLazy(func() string {
		entries := GenerateAliases(parent, exec.LookPath)
		skip := completion.RegisteredAliases()
		return FormatFish(entries) + FormatFishCompletions(entries, skip...)
	})
}
