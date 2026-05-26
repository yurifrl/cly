//go:build darwin

package notify

import _ "embed"

//go:embed assets/cly-notifier.app.tar.gz
var notifierBundle []byte

// notifierBundleAvailable reports whether the embedded bundle is a real
// build artifact rather than the committed placeholder. The placeholder
// is intentionally tiny (a few bytes) so we can detect it without parsing.
func notifierBundleAvailable() bool {
	return len(notifierBundle) > 1024
}
