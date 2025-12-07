package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Loader represents a loading animation
type Loader struct {
	message string
	done    chan bool
	debug   bool
}

// NewLoader creates a new loader with the given message
func NewLoader(message string, debug bool) *Loader {
	return &Loader{
		message: message,
		done:    make(chan bool),
		debug:   debug,
	}
}

// Start begins the loading animation
func (l *Loader) Start() {
	if l.debug {
		return // Don't show loader in debug mode
	}

	go func() {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-l.done:
				return
			default:
				fmt.Printf("\r%s %s", spinner[i%len(spinner)], l.message)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop ends the loading animation
func (l *Loader) Stop() {
	if l.debug {
		return // Nothing to stop in debug mode
	}

	l.done <- true
	fmt.Print("\r") // Clear the line
}

// RunCommandWithLoader executes a command with optional loader animation
func RunCommandWithLoader(cmd *exec.Cmd, message string, debug bool) error {
	if debug {
		// In debug mode, show all output
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("🔧 %s...\n", message)
		return cmd.Run()
	}

	// In non-debug mode, hide output and show loader
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	loader := NewLoader(message, debug)
	loader.Start()
	defer loader.Stop()

	return cmd.Run()
}

// RunCommandWithLoaderAndContext executes a command with context, loader, and debug support
func RunCommandWithLoaderAndContext(ctx context.Context, cmd *exec.Cmd, message string, debug bool) error {
	if debug {
		// In debug mode, show all output
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("🔧 %s...\n", message)
		return cmd.Run()
	}

	// In non-debug mode, hide output and show loader
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	loader := NewLoader(message, debug)
	loader.Start()
	defer loader.Stop()

	// Use context if provided
	if ctx != nil {
		return cmd.Run()
	}

	return cmd.Run()
}
