package bundle

// Bundler defines the interface for package managers.
type Bundler interface {
	Name() string
	DefaultFile() string
	CheckDeps() error
	Sync(bundleFile string, verbose bool) error
	Check(bundleFile string) error
	Cleanup(bundleFile string, verbose bool) error
}
