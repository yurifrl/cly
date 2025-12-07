package builder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/NSXBet/nsx-cli/shared/interact"
)

// isYAMLFile checks if a file is a YAML/YML file based on its extension
func isYAMLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yaml" || ext == ".yml"
}

// mergeYAMLFiles merges two YAML contents together, preserving template file order
func mergeYAMLFiles(existing, new string) (string, error) {
	var existingData, newData yaml.Node

	// Parse existing YAML with order preservation
	if err := yaml.Unmarshal([]byte(existing), &existingData); err != nil {
		return "", fmt.Errorf("failed to parse existing YAML: %w", err)
	}

	// Parse new YAML with order preservation
	if err := yaml.Unmarshal([]byte(new), &newData); err != nil {
		return "", fmt.Errorf("failed to parse new YAML: %w", err)
	}

	// Merge the nodes, preserving template order
	merged := mergeYAMLNodes(&existingData, &newData)

	// Convert back to YAML with 2-space indentation
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(merged); err != nil {
		return "", fmt.Errorf("failed to encode merged YAML: %w", err)
	}

	return buf.String(), nil
}

// mergeYAMLNodes recursively merges two YAML nodes, preserving template order
func mergeYAMLNodes(existing, template *yaml.Node) *yaml.Node {
	// If template is nil, return existing
	if template == nil {
		return existing
	}

	// If existing is nil, return template
	if existing == nil {
		return template
	}

	// If nodes are different types, prefer template
	if existing.Kind != template.Kind {
		return template
	}

	// Handle different node types
	switch template.Kind {
	case yaml.DocumentNode:
		// For documents, merge the content
		if len(template.Content) > 0 && len(existing.Content) > 0 {
			merged := &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{mergeYAMLNodes(existing.Content[0], template.Content[0])},
			}
			return merged
		}
		return template

	case yaml.MappingNode:
		// Create a new mapping node starting with template structure
		result := &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: make([]*yaml.Node, 0),
		}

		// Create maps for quick lookup
		existingMap := make(map[string]*yaml.Node)
		templateMap := make(map[string]*yaml.Node)

		// Build existing map
		for i := 0; i < len(existing.Content); i += 2 {
			if i+1 < len(existing.Content) {
				key := existing.Content[i].Value
				existingMap[key] = existing.Content[i+1]
			}
		}

		// Build template map and preserve order
		for i := 0; i < len(template.Content); i += 2 {
			if i+1 < len(template.Content) {
				key := template.Content[i].Value
				templateMap[key] = template.Content[i+1]
			}
		}

		// Iterate through template keys in order
		for i := 0; i < len(template.Content); i += 2 {
			if i+1 < len(template.Content) {
				keyNode := template.Content[i]
				templateValue := template.Content[i+1]
				key := keyNode.Value

				// Add key
				result.Content = append(result.Content, keyNode)

				// Merge value if exists in existing, otherwise use template value
				if existingValue, exists := existingMap[key]; exists {
					mergedValue := mergeYAMLNodes(existingValue, templateValue)
					result.Content = append(result.Content, mergedValue)
				} else {
					result.Content = append(result.Content, templateValue)
				}
			}
		}

		// Add any keys from existing that aren't in template (append at end)
		for i := 0; i < len(existing.Content); i += 2 {
			if i+1 < len(existing.Content) {
				keyNode := existing.Content[i]
				value := existing.Content[i+1]
				key := keyNode.Value

				if _, existsInTemplate := templateMap[key]; !existsInTemplate {
					result.Content = append(result.Content, keyNode)
					result.Content = append(result.Content, value)
				}
			}
		}

		return result

	case yaml.SequenceNode:
		// For sequences, merge by appending existing items to template
		result := &yaml.Node{
			Kind:    yaml.SequenceNode,
			Content: make([]*yaml.Node, 0),
		}

		// Start with template items
		result.Content = append(result.Content, template.Content...)

		// Add existing items that aren't duplicates
		for _, existingItem := range existing.Content {
			isDuplicate := false
			for _, templateItem := range template.Content {
				if nodesEqual(existingItem, templateItem) {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				result.Content = append(result.Content, existingItem)
			}
		}

		return result

	default:
		// For scalar values, prefer template
		return template
	}
}

// nodesEqual checks if two YAML nodes are equal
func nodesEqual(a, b *yaml.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Kind != b.Kind || a.Value != b.Value {
		return false
	}
	if len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !nodesEqual(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}

// writeFileWithCollisionHandling handles file collisions based on file type
func writeFileWithCollisionHandling(outPath, content string) error {
	// Check if file already exists
	if existingData, err := os.ReadFile(outPath); err == nil {
		// File exists, handle collision
		existing := string(existingData)

		if isYAMLFile(outPath) {
			// For YAML files, merge the content
			interact.Info("🔀 Merging YAML file: %s", filepath.Base(outPath))
			merged, err := mergeYAMLFiles(existing, content)
			if err != nil {
				return fmt.Errorf("failed to merge YAML files: %w", err)
			}
			content = merged
		} else {
			// For non-YAML files, append the content
			interact.Info("➕ Appending to file: %s", filepath.Base(outPath))
			content = existing + "\n\n" + content
		}
	}

	// Write the final content
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outPath, err)
	}

	return nil
}
