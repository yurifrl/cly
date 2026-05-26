// Package notify generators.
//
// Run `go generate ./pkg/notify/...` on darwin to build the embedded
// cly-notifier.app bundle. Non-darwin hosts skip the script.
package notify

//go:generate ./swift/build.sh
