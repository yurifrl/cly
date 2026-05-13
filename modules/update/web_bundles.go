package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yurifrl/cly/pkg/style"
)

// webBundle describes a frontend directory that must be built into dist/
// so Go can embed it.
type webBundle struct {
	dir  string // relative to sourceDir
	name string // human label for logs
}

// knownWebBundles lists all embeddable frontends in the repo.
// Add new entries here when a module adds a web/ directory.
var knownWebBundles = []webBundle{
	{dir: "modules/diff2/web", name: "diff2"},
}

// buildWebBundles runs `npm install && npm run build` in each known
// frontend directory so //go:embed picks up the latest dist/.
// Missing npm or a missing package.json both skip the bundle with a
// warning rather than failing the whole build.
func buildWebBundles(sourceDir string) error {
	npm, err := exec.LookPath("npm")
	if err != nil {
		fmt.Printf("%s npm not found on PATH; skipping web bundle builds\n",
			style.YellowStyle.Render("⚠️"))
		return nil
	}

	for _, b := range knownWebBundles {
		full := filepath.Join(sourceDir, b.dir)
		if _, err := os.Stat(filepath.Join(full, "package.json")); err != nil {
			fmt.Printf("%s skipping %s bundle (no package.json at %s)\n",
				style.YellowStyle.Render("⚠️"), b.name, full)
			continue
		}
		fmt.Printf("%s Building %s web bundle in %s\n",
			style.BlueStyle.Render("⚡"), b.name, b.dir)

		// npm install (idempotent, cheap with lockfile)
		install := exec.Command(npm, "install", "--silent", "--no-fund", "--no-audit")
		install.Dir = full
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("%s: npm install: %w", b.name, err)
		}

		// npm run build
		build := exec.Command(npm, "run", "build")
		build.Dir = full
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return fmt.Errorf("%s: npm run build: %w", b.name, err)
		}
	}
	return nil
}
