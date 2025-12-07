package builder

import "github.com/NSXBet/nsx-cli/shared/blueprint/module"

// Options hold wizard answers and computed data
type Options struct {
	ProjectName           string
	ProjectNamePascalCase string
	ProjectNameSnakeCase  string
	ProjectNameTitle      string
	Team                  string
	Debug                 bool
	ProxyURL              string

	SelectedModules []module.Module
	Imports         map[string]module.ImportInfo
}

// ProjectArguments contains command-line arguments for project generation
type ProjectArguments struct {
	ProjectName string
	Team        string
	Debug       bool
	ProxyURL    string
}
