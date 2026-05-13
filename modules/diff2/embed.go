package diff2

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var webFS embed.FS

// Embedded returns the built frontend rooted at the dist directory.
// When the bundle has not been built, the placeholder in web/dist/index.html
// is served instead, informing the user how to build.
func Embedded() (fs.FS, error) {
	return fs.Sub(webFS, "web/dist")
}
