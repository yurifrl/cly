// Package diff2 serves a local web UI for reviewing the working-tree
// git diff and attaching bd beads to changed files.
//
// Naming: "diff2" avoids collision with the existing modules/beads/ TUI
// form and other diff-adjacent helpers. Rename to "diff" later if desired.
package diff2

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// Flags bound by the cobra command.
type flags struct {
	port   int
	noOpen bool
}

// Register attaches the diff2 command to parent.
func Register(parent *cobra.Command) {
	f := &flags{}

	cmd := &cobra.Command{
		Use:   "diff2",
		Short: "Review working-tree diff in a local web UI, capture beads",
		Long: `cly diff2 spawns a local HTTP server bound to 127.0.0.1,
opens your default browser, and shows ` + "`git diff HEAD`" + ` of the
working tree. Press 'n' on a file to open the New Bead modal, which
shells out to ` + "`bd create`" + `.

Single binary: React frontend embedded via go:embed.
Ctrl+C to stop the server and exit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, f)
		},
	}

	cmd.Flags().IntVar(&f.port, "port", 0, "bind port (0 = pick free port)")
	cmd.Flags().BoolVar(&f.noOpen, "no-open", false, "do not open browser, print URL only")

	parent.AddCommand(cmd)
}

// run is the command entrypoint.
func run(cmd *cobra.Command, f *flags) error {
	// Fail fast if not in a git repo — clearer UX than waiting for /api/diff.
	if err := IsRepo(execGit{}); err != nil {
		return err
	}

	web, err := webAssets()
	if err != nil {
		return fmt.Errorf("diff2: web assets: %w", err)
	}

	deps := RealDeps(web)
	handler := NewServer(deps)

	l, port, err := Listen(f.port)
	if err != nil {
		return fmt.Errorf("diff2: listen: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Fprintf(cmd.OutOrStdout(), "cly diff2 serving at %s\n", url)
	fmt.Fprintln(cmd.OutOrStdout(), "press Ctrl+C to stop")

	srv := &http.Server{Handler: handler}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	if !f.noOpen {
		if err := OpenBrowser(url); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warn: could not open browser: %v\n", err)
		}
	}

	// Wait for signal or serve error.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		fmt.Fprintln(cmd.OutOrStdout(), "\nshutting down…")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// webAssets returns the fs.FS holding the built React bundle.
func webAssets() (fs.FS, error) {
	return Embedded()
}
