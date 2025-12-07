package module

import (
	"io/fs"
)

// Module defines extension behaviour for generation.
type Module interface {
	ID() string           // identifier shown in UI
	Description() string  // human description
	Type() ModuleType     // category type for organization
	TemplateFS() fs.FS    // fs containing templates
	TemplateRoot() string // path inside fs to walk
}

// ModuleWithImports is a module that has additional imports.
type ModuleWithImports interface {
	Module
	Imports(projectName string) map[string]ImportInfo
}

type ImportInfo struct {
	FxImports []string // fx module imports
	Paths     []string // import paths to add to the project
}

// AllModules lists every available module.
var AllModules []Module

// RegisterModule adds a module to the list
func RegisterModule(module Module) {
	AllModules = append(AllModules, module)
}

// FindModule returns module by ID or nil.
func FindModule(id string) Module {
	for _, m := range AllModules {
		if m.ID() == id {
			return m
		}
	}
	return nil
}
