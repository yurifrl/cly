package builder

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// createBaseFolder sets up base folder
func createBaseFolder(destDir string) error {
	// Create destination directory
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return nil
}

// processEmbeddedDir processes templates from embedded filesystem
func processEmbeddedDir(vfs fs.FS, root, destDir string, opts *Options) error {
	return fs.WalkDir(vfs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walkdir error for path %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// Only render the path as a template if it contains template variables
		renderedRel := rel
		if strings.Contains(rel, "{{") {
			renderedRel, err = renderTemplate(rel, opts)
			if err != nil {
				return fmt.Errorf("failed to render template path %s: %w", rel, err)
			}
		}

		// Remove .tmpl suffix
		renderedRel = strings.TrimSuffix(renderedRel, ".tmpl")

		// Convert special directory names to hidden directories
		renderedRel = strings.Replace(renderedRel, "github/", ".github/", 1)
		renderedRel = strings.Replace(renderedRel, "gitignore", ".gitignore", 1)
		renderedRel = strings.Replace(renderedRel, "gitkeep", ".gitkeep", 1)
		renderedRel = strings.Replace(renderedRel, "mockery.yaml", ".mockery.yaml", 1)
		renderedRel = strings.Replace(renderedRel, "editorconfig", ".editorconfig", 1)

		outPath := filepath.Join(destDir, renderedRel)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outPath, err)
		}
		data, err := fs.ReadFile(vfs, path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		content, err := renderTemplate(string(data), opts)
		if err != nil {
			return fmt.Errorf("failed to render template content for %s: %w", path, err)
		}
		if err := writeFileWithCollisionHandling(outPath, content); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outPath, err)
		}
		return nil
	})
}

// renderTemplate processes template content with provided options
func renderTemplate(tpl string, opts *Options) (string, error) {
	t, err := template.New("tpl").Funcs(template.FuncMap{
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"snakecase":  ToSnakeCase,
		"kebabcase":  ToKebabCase,
		"camelcase":  ToCamelCase,
		"pascalcase": ToPascalCase,
	}).Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, opts); err != nil {
		return "", err
	}

	// Unescape GitHub Actions interpolations: \{\{ becomes {{
	result := strings.ReplaceAll(buf.String(), "\\{\\{", "{{")
	result = strings.ReplaceAll(result, "\\}\\}", "}}")

	return result, nil
}
