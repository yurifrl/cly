// Package cmux provides a thin client for the cmux terminal multiplexer CLI.
// It is only active when running inside a cmux session (CMUX_WORKSPACE_ID is set).
package cmux

import (
	"context"
	"os"
	"os/exec"
)

// Available returns true when the process is running inside a cmux session.
func Available() bool {
	return os.Getenv("CMUX_WORKSPACE_ID") != ""
}

// Notify sends a desktop notification via `cmux notify --title … --body …`.
// It is a no-op when not inside a cmux session.
func Notify(ctx context.Context, title, body string) error {
	if !Available() {
		return nil
	}
	return exec.CommandContext(ctx, "cmux", "notify", "--title", title, "--body", body).Run()
}

// StatusOption configures an optional flag for SetStatus.
type StatusOption func(*statusOptions)

type statusOptions struct {
	icon  string
	color string
}

// WithIcon sets the icon name for a status entry (e.g. "checkmark", "hammer").
func WithIcon(name string) StatusOption {
	return func(o *statusOptions) { o.icon = name }
}

// WithColor sets the hex color for a status entry (e.g. "#196F3D").
func WithColor(hex string) StatusOption {
	return func(o *statusOptions) { o.color = hex }
}

// SetStatus sets a sidebar status key via `cmux set-status <key> <value>`.
// It is a no-op when not inside a cmux session.
func SetStatus(ctx context.Context, key, value string, opts ...StatusOption) error {
	if !Available() {
		return nil
	}
	o := &statusOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := []string{"set-status", key, value}
	if o.icon != "" {
		args = append(args, "--icon", o.icon)
	}
	if o.color != "" {
		args = append(args, "--color", o.color)
	}
	return exec.CommandContext(ctx, "cmux", args...).Run()
}

// ClearStatus removes a sidebar status key via `cmux clear-status <key>`.
// It is a no-op when not inside a cmux session.
func ClearStatus(ctx context.Context, key string) error {
	if !Available() {
		return nil
	}
	return exec.CommandContext(ctx, "cmux", "clear-status", key).Run()
}
