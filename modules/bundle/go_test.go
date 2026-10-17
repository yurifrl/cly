package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGoPkg(t *testing.T) {
	tests := []struct {
		spec     string
		expected string
		version  string
	}{
		{"mvdan.cc/gofumpt", "mvdan.cc/gofumpt", ""},
		{"mvdan.cc/gofumpt@latest", "mvdan.cc/gofumpt", "latest"},
		{"golang.org/x/tools/cmd/goimports@v0.24.0", "golang.org/x/tools/cmd/goimports", "v0.24.0"},
		{"github.com/rakyll/hey@v0.1.12", "github.com/rakyll/hey", "v0.1.12"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			spec, version := parseGoPkg(tt.spec)
			assert.Equal(t, tt.expected, spec)
			assert.Equal(t, tt.version, version)
		})
	}
}

func TestMatchGoTool(t *testing.T) {
	tool := goTool{
		binary:  "/Users/x/go/bin/goimports",
		module:  "golang.org/x/tools",
		version: "v0.24.0",
	}
	moduleTool := goTool{
		binary:  "/Users/x/go/bin/gofumpt",
		module:  "mvdan.cc/gofumpt",
		version: "v0.7.0",
	}

	tests := []struct {
		spec     string
		tool     goTool
		expected bool
	}{
		// Module root tools: spec is the module path.
		{"mvdan.cc/gofumpt", moduleTool, true},
		{"mvdan.cc/gofumpt@v0.7.0", moduleTool, true},
		// Main packages inside a multi-package module: binary name match.
		{"golang.org/x/tools/cmd/goimports", tool, true},
		{"golang.org/x/tools/cmd/goimports@v0.24.0", tool, true},
		// Wrong command inside the module.
		{"golang.org/x/tools/cmd/stringer", tool, false},
		// Wrong module.
		{"mvdan.cc/sh", tool, false},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			spec, _ := parseGoPkg(tt.spec)
			assert.Equal(t, tt.expected, matchGoTool(spec, tt.tool))
		})
	}
}

func TestFindGoTool(t *testing.T) {
	tools := []goTool{
		{binary: "/Users/x/go/bin/goimports", module: "golang.org/x/tools", version: "v0.24.0"},
		{binary: "/Users/x/go/bin/gofumpt", module: "mvdan.cc/gofumpt", version: "v0.7.0"},
	}

	t.Run("finds module-root tool", func(t *testing.T) {
		tool, found := findGoTool("mvdan.cc/gofumpt", tools)
		assert.True(t, found)
		assert.Equal(t, "v0.7.0", tool.version)
	})

	t.Run("finds main-package tool", func(t *testing.T) {
		tool, found := findGoTool("golang.org/x/tools/cmd/goimports@v0.24.0", tools)
		assert.True(t, found)
		assert.Equal(t, "v0.24.0", tool.version)
	})

	t.Run("misses unknown spec", func(t *testing.T) {
		_, found := findGoTool("github.com/nobody/nowhere", tools)
		assert.False(t, found)
	})
}
