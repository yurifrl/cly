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
	flagLocal  bool
	flagIDE    string
	flagDetach bool
	flagRm     bool
)

// Register adds the agents command tree to the parent.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Sync .agents configs to IDE directories",
		Long:  "Synchronize .agents configurations to IDE-specific directories (.claude, .opencode, .crush).",
	}

	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be synced without writing")
	cmd.PersistentFlags().BoolVar(&flagGlobal, "global", false, "Sync only global configs")
	cmd.PersistentFlags().StringVarP(&flagIDE, "ide", "i", "", "Sync only this IDE (claude, opencode, crush)")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run agents sync (foreground watcher by default)",
		RunE:  runStart,
	}
	runCmd.Flags().BoolVarP(&flagDetach, "detach", "d", false, "Run as background daemon with socket")
	runCmd.Flags().BoolVar(&flagRm, "rm", false, "One-shot sync and exit")

	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Create agents.yaml config file",
		RunE:  runConfigure,
	}
	configureCmd.Flags().BoolVar(&flagLocal, "local", false, "Create .agents/agents.yaml in current directory instead of global")

	cmd.AddCommand(
		runCmd,
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
		configureCmd,
	)

	parent.AddCommand(cmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	if flagRm {
		return doSync()
	}
	if flagDetach {
		if err := doSync(); err != nil {
			return err
		}
		return startDaemon()
	}
	// Default: foreground watcher, no socket
	if err := doSync(); err != nil {
		return err
	}
	return startWatcher()
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

const defaultConfigContent = "ides:\n  - claude\n  - opencode\n"

func runConfigure(cmd *cobra.Command, args []string) error {
	var dir string
	if flagLocal {
		dir = ".agents"
	} else {
		dir = filepath.Join(homeDir(), ".agents")
	}

	path := filepath.Join(dir, ConfigFileName)

	// If file exists, print it and exit
	if data, err := os.ReadFile(path); err == nil {
		fmt.Printf("%s already exists:\n%s", path, string(data))
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(defaultConfigContent), 0644); err != nil {
		return err
	}

	fmt.Printf("created %s\n", path)
	return nil
}

func doSync() error {
	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)

		cfg, ides := loadConfigForScope(sourceDirs)
		if cfg == nil {
			continue // no config = no sync for this scope
		}

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

	// Build merged reverse map from all scopes/IDEs
	reverseMap := make(map[string]string)

	// syncAndCollectPlans does a forward sync and returns the merged reverse map
	syncFn := func() (*ReconcileResult, error) {
		combined := &ReconcileResult{}

		for _, scope := range syncScopes() {
			sourceDirs := ResolveSourceDirs(scope.global)
			cfg, ides := loadConfigForScope(sourceDirs)
			if cfg == nil {
				continue
			}

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
					// Merge reverse map
					for t, s := range plan.ReverseMap {
						reverseMap[t] = s
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

	// Set up watcher — forward sync callback will be set after watcher is created
	watcher, err := NewWatcher(nil)
	if err != nil {
		return err
	}
	watcher.onSync = func() {
		watcher.SuppressBriefly()
		result, err := syncFn()
		if err == nil && result.Written > 0 {
			fmt.Printf("synced: %d written, %d skipped\n", result.Written, result.Skipped)
		}
	}

	// Set up reverse sync callback
	watcher.SetReverseSync(func(path string) {
		watcher.SuppressBriefly()
		written, err := ReverseReconcile(path, reverseMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reverse sync %s: %v\n", path, err)
			return
		}
		if written {
			fmt.Printf("reverse synced: %s\n", filepath.Base(path))
		}
	})

	// Watch source dirs
	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)
		for _, dir := range sourceDirs {
			if _, err := os.Stat(dir); err == nil {
				watchRecursive(watcher, dir)
			}
		}
	}

	// Watch target dirs for bidirectional sync
	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)
		cfg, ides := loadConfigForScope(sourceDirs)
		if cfg == nil {
			continue
		}
		for _, ideName := range ides {
			ide := GetIDEDef(ideName)
			if ide == nil {
				continue
			}
			targetBase := resolveTarget(ide, scope.global)
			if _, err := os.Stat(targetBase); err == nil {
				watchRecursiveTarget(watcher, targetBase)
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

func startWatcher() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	reverseMap := make(map[string]string)

	syncFn := func() (*ReconcileResult, error) {
		combined := &ReconcileResult{}
		for _, scope := range syncScopes() {
			sourceDirs := ResolveSourceDirs(scope.global)
			cfg, ides := loadConfigForScope(sourceDirs)
			if cfg == nil {
				continue
			}
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
					for t, s := range plan.ReverseMap {
						reverseMap[t] = s
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

	watcher, err := NewWatcher(nil)
	if err != nil {
		return err
	}
	watcher.onSync = func() {
		watcher.SuppressBriefly()
		result, err := syncFn()
		if err == nil && result.Written > 0 {
			fmt.Printf("synced: %d written, %d skipped\n", result.Written, result.Skipped)
		}
	}
	watcher.SetReverseSync(func(path string) {
		watcher.SuppressBriefly()
		written, err := ReverseReconcile(path, reverseMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reverse sync %s: %v\n", path, err)
			return
		}
		if written {
			fmt.Printf("reverse synced: %s\n", filepath.Base(path))
		}
	})

	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)
		for _, dir := range sourceDirs {
			if _, err := os.Stat(dir); err == nil {
				watchRecursive(watcher, dir)
			}
		}
	}
	for _, scope := range syncScopes() {
		sourceDirs := ResolveSourceDirs(scope.global)
		cfg, ides := loadConfigForScope(sourceDirs)
		if cfg == nil {
			continue
		}
		for _, ideName := range ides {
			ide := GetIDEDef(ideName)
			if ide == nil {
				continue
			}
			targetBase := resolveTarget(ide, scope.global)
			if _, err := os.Stat(targetBase); err == nil {
				watchRecursiveTarget(watcher, targetBase)
			}
		}
	}

	fmt.Println("watching for changes (ctrl+c to stop)")
	return watcher.Run(ctx)
}

// watchRecursive adds a directory and all its subdirectories to the watcher (source dirs).
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

// watchRecursiveTarget adds a directory and all its subdirectories as target dirs.
func watchRecursiveTarget(w *Watcher, root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			w.AddTargetDir(path)
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

// loadConfigForScope finds and parses config for a set of source dirs.
// Returns nil if no config found (no config = no sync).
func loadConfigForScope(sourceDirs []string) (*Config, []string) {
	configPath := FindConfigFile(sourceDirs)
	if configPath == "" {
		return nil, nil
	}

	cfg, _ := ParseConfig(configPath)
	if cfg == nil {
		return nil, nil
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
