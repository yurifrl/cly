// Package cmux provides a thin client for the cmux terminal multiplexer CLI.
// It is only active when running inside a cmux session (CMUX_WORKSPACE_ID is set).
package cmux

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yurifrl/cly/pkg/envs"
)

// Available returns true when the process is running inside a cmux session.
func Available() bool {
	return envs.InCmux()
}

// BinaryAvailable returns true when the `cmux` binary is on PATH. Use this for
// commands like `cmux notify` that work outside a cmux session (they target
// the running cmux app directly).
func BinaryAvailable() bool {
	_, err := exec.LookPath("cmux")
	return err == nil
}

// Notify sends a desktop notification via `cmux notify --title … --body …`.
// Works as long as the cmux binary is on PATH and the cmux app is running —
// does not require being inside a cmux session.
func Notify(ctx context.Context, title, body string) error {
	if !BinaryAvailable() {
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

// SetDescription writes a workspace description slot via
// `cmux workspace-action --action set-description`. The custom sidebar reads
// this as `w.description`; workspace must be a UUID, ref, or index. The flag
// targets the workspace explicitly, so this works from any shell — inside
// cmux surfaces or not.
func SetDescription(ctx context.Context, workspace, text string) error {
	if workspace == "" {
		return fmt.Errorf("set-description: empty workspace")
	}
	out, err := exec.CommandContext(ctx, "cmux", "workspace-action",
		"--workspace", workspace, "--action", "set-description",
		"--description", text).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmux set-description %s: %w: %s", workspace, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RightSidebarSet switches the right sidebar to a custom sidebar by name
// (files under ~/.config/cmux/sidebars). `--no-focus` keeps the user's
// current focus; the switch is idempotent, so it is safe on every start.
func RightSidebarSet(ctx context.Context, name string) error {
	out, err := exec.CommandContext(ctx, "cmux", "right-sidebar", "set", "custom", name, "--no-focus").CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmux right-sidebar set custom %s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RightSidebarShow makes the right sidebar visible without moving focus —
// the "open the visualization" verb for commands that push into it.
func RightSidebarShow(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "cmux", "right-sidebar", "show").CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmux right-sidebar show: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
