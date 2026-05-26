//go:build !darwin

package notify

import "context"

// NativeMacOSNotifier is a stub on non-darwin platforms.
type NativeMacOSNotifier struct{}

func NewNativeMacOSNotifier(_ context.Context) *NativeMacOSNotifier {
	return &NativeMacOSNotifier{}
}

func (n *NativeMacOSNotifier) Send(_ context.Context, _ Notification) error { return nil }
func (n *NativeMacOSNotifier) Available() bool                              { return false }
func (n *NativeMacOSNotifier) Close() error                                 { return nil }
