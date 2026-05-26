//go:build !darwin

package notify

var notifierBundle []byte

func notifierBundleAvailable() bool { return false }
