package mysql

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/NSXBet/nsx-cli/shared/blueprint/module"
)

// mysql module implementation
func init() {
	module.RegisterModule(NewModule())
}

type Module struct {
	templateFS fs.FS
}

//go:embed files
var templateFS embed.FS

// NewModule creates a new mysql module with the provided FS
func NewModule() module.Module {
	return Module{templateFS: templateFS}
}

func (m Module) ID() string              { return "mysql" }
func (m Module) Description() string     { return "MySQL database support" }
func (m Module) Type() module.ModuleType { return module.DatabaseModule }
func (m Module) TemplateFS() fs.FS       { return m.templateFS }
func (m Module) TemplateRoot() string    { return filepath.Join("files") }
func (m Module) Imports(projectName string) map[string]module.ImportInfo {
	return map[string]module.ImportInfo{
		"all": {
			FxImports: []string{"database.Module"},
			Paths:     []string{fmt.Sprintf("github.com/NSXBet/%s/infra/database", projectName)},
		},
	}
}
