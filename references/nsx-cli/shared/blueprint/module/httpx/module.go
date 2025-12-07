package httpx

import (
	"embed"
	"io/fs"
	"path/filepath"

	"github.com/NSXBet/nsx-cli/shared/blueprint/module"
)

// httpx module implementation
func init() {
	module.RegisterModule(NewModule())
}

type Module struct {
	templateFS embed.FS
}

//go:embed files
var templateFS embed.FS

// NewModule creates a new httpx module with the provided FS
func NewModule() module.Module {
	return Module{templateFS: templateFS}
}

func (m Module) ID() string              { return "httpx" }
func (m Module) Description() string     { return "HTTPX http server support" }
func (m Module) Type() module.ModuleType { return module.HandlerModule }
func (m Module) TemplateFS() fs.FS       { return m.templateFS }
func (m Module) TemplateRoot() string    { return filepath.Join("files") }
