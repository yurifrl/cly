package diff2

import (
	"io/fs"
	"testing"
)

// TestEmbedded asserts the dist placeholder exists so the binary is always
// servable. Real `vite build` output overwrites these files.
func TestEmbedded_HasIndex(t *testing.T) {
	sub, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		t.Errorf("web/dist/index.html missing: %v", err)
	}
}
