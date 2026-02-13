package agents

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	flagDryRun bool
	flagGlobal bool
	flagIDE    string
)

// Register adds the agents command tree to the parent.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Sync .agents configs to IDE directories",
		Long:  "Synchronize .agents configurations to IDE-specific directories (.claude, .opencode, .crush).",
		RunE:  runDefault,
	}

	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be synced without writing")
	cmd.PersistentFlags().BoolVar(&flagGlobal, "global", false, "Sync only global configs")
	cmd.PersistentFlags().StringVarP(&flagIDE, "ide", "i", "", "Sync only this IDE (claude, opencode, crush)")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "sync",
			Short: "One-shot sync",
			RunE:  runSync,
		},
		&cobra.Command{
			Use:   "daemon",
			Short: "Start foreground daemon with file watching",
			RunE:  runDaemon,
		},
		&cobra.Command{
			Use:   "status",
			Short: "Query running daemon status",
			RunE:  runStatus,
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop running daemon",
			RunE:  runStop,
		},
	)

	parent.AddCommand(cmd)
}

// runDefault: if daemon running → status; else → one sync + start watching
func runDefault(cmd *cobra.Command, args []string) error {
	client := NewDaemonClient(SocketPath)
	if client.IsRunning() {
		return runStatus(cmd, args)
	}

	// One sync first
	if err := doSync(); err != nil {
		return err
	}

	// Then start daemon
	return startDaemon()
}

func runSync(cmd *cobra.Command, args []string) error {
	return doSync()
}

func runDaemon(cmd *cobra.Command, args []string) error {
	// One sync first
	if err := doSync(); err != nil {
		return err
	}
	return startDaemon()
}

func runStatus(cmd *cobra.Command, args []string) error {
	client := NewDaemonClient(SocketPath)
	resp, err := client.Status()
	if err != nil {
		return fmt.Errorf("daemon not running")
	}
	if resp.OK {
		fmt.Println("daemon running")
		if data, ok := resp.Data.(map[string]interface{}); ok {
			for k, v := range data {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	}
	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	client := NewDaemonClient(SocketPath)
	if err := client.Stop(); err != nil {
		return err
	}
	fmt.Println("daemon stopped")
	return nil
}

func doSync() error {
	cfg, ides := loadConfig()

	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)
		for _, sourceDir := range sourceDirs {
			if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
				continue
			}

			for _, ideName := range ides {
				ide := GetIDEDef(ideName)
				if ide == nil {
					continue
				}

				targetBase := resolveTarget(ide, scope.global)
				plan, err := Discover(sourceDir, ide, targetBase)
				if err != nil {
					fmt.Fprintf(os.Stderr, "discover %s/%s: %v\n", sourceDir, ideName, err)
					continue
				}

				if len(plan.Items) == 0 {
					continue
				}

				result, err := Reconcile(plan, flagDryRun)
				if err != nil {
					fmt.Fprintf(os.Stderr, "reconcile %s/%s: %v\n", sourceDir, ideName, err)
					continue
				}

				prefix := ""
				if flagDryRun {
					prefix = "[dry-run] "
				}
				if result.Written > 0 || flagDryRun {
					fmt.Printf("%s%s → %s: %d written, %d skipped\n",
						prefix, filepath.Base(sourceDir), ideName, result.Written, result.Skipped)
				}
			}
		}
	}

	_ = cfg // used for IDE list
	return nil
}

func startDaemon() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	syncFn := func() (*ReconcileResult, error) {
		combined := &ReconcileResult{}
		_, ides := loadConfig()

		for _, scope := range syncScopes() {
			sourceDirs := ResolveSourceDirs(scope.global)
			for _, sourceDir := range sourceDirs {
				if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
					continue
				}
				for _, ideName := range ides {
					ide := GetIDEDef(ideName)
					if ide == nil {
						continue
					}
					targetBase := resolveTarget(ide, scope.global)
					plan, err := Discover(sourceDir, ide, targetBase)
					if err != nil {
						continue
					}
					result, err := Reconcile(plan, false)
					if err != nil {
						continue
					}
					combined.Written += result.Written
					combined.Skipped += result.Skipped
				}
			}
		}
		return combined, nil
	}

	daemon := NewDaemon(SocketPath, syncFn)

	// Set up watcher
	watcher, err := NewWatcher(func() {
		result, err := syncFn()
		if err == nil && result.Written > 0 {
			fmt.Printf("synced: %d written, %d skipped\n", result.Written, result.Skipped)
		}
	})
	if err != nil {
		return err
	}

	// Watch source dirs
	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)
		for _, dir := range sourceDirs {
			if _, err := os.Stat(dir); err == nil {
				watchRecursive(watcher, dir)
			}
		}
	}

	fmt.Printf("daemon listening on %s\n", SocketPath)

	// Run watcher and daemon concurrently
	errCh := make(chan error, 2)
	go func() { errCh <- watcher.Run(ctx) }()
	go func() { errCh <- daemon.Run(ctx) }()

	return <-errCh
}

// watchRecursive adds a directory and all its subdirectories to the watcher.
func watchRecursive(w *Watcher, root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			w.Add(path)
		}
		return nil
	})
}

type syncScope struct {
	global bool
}

func syncScopes() []syncScope {
	if flagGlobal {
		return []syncScope{{global: true}}
	}
	return []syncScope{{global: false}, {global: true}}
}

func loadConfig() (*Config, []string) {
	sourceDirs := ResolveSourceDirs(flagGlobal)
	configPath := FindConfigFile(sourceDirs)

	var cfg *Config
	if configPath != "" {
		cfg, _ = ParseConfig(configPath)
	}
	if cfg == nil {
		cfg = &Config{IDEs: DefaultIDEs}
	}

	ides := cfg.IDEs
	if flagIDE != "" {
		ides = []string{flagIDE}
	}

	return cfg, ides
}

func resolveTarget(ide *IDEDef, global bool) string {
	if global {
		return ide.GlobalDir
	}
	return ide.LocalDir
}
