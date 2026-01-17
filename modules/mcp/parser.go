package mcp

import (
	"encoding/json"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// detectFormat returns the format based on file extension
func detectFormat(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".jsonc":
		return "jsonc"
	default:
		return "json"
	}
}

// ParseMCPFile parses MCP definitions from file in any supported format
func ParseMCPFile(path string, data []byte) (map[string]MCP, error) {
	format := detectFormat(path)

	switch format {
	case "yaml":
		return parseYAML(data)
	case "jsonc":
		return parseJSONC(data)
	default:
		return parseJSON(data)
	}
}

func parseJSON(data []byte) (map[string]MCP, error) {
	// Try flat format first
	var mcps map[string]MCP
	if err := json.Unmarshal(data, &mcps); err == nil {
		// Check if it's actually mcpServers wrapper format
		if len(mcps) == 1 {
			if _, hasMcpServers := mcps["mcpServers"]; hasMcpServers {
				var wrapped struct {
					MCPServers map[string]MCP `json:"mcpServers"`
				}
				if err := json.Unmarshal(data, &wrapped); err == nil {
					return wrapped.MCPServers, nil
				}
			}
		}
		return mcps, nil
	}

	// Try mcpServers wrapper format as fallback
	var wrapped struct {
		MCPServers map[string]MCP `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.MCPServers, nil
}

func parseJSONC(data []byte) (map[string]MCP, error) {
	// Strip comments and trailing commas from JSONC
	cleaned := stripJSONComments(data)
	cleaned = stripTrailingCommas(cleaned)
	return parseJSON(cleaned)
}

func parseYAML(data []byte) (map[string]MCP, error) {
	var mcps map[string]MCP
	err := yaml.Unmarshal(data, &mcps)
	return mcps, err
}

// stripJSONComments removes // and /* */ style comments from JSONC
// This is a simple implementation - for production use, consider a proper JSONC parser
func stripJSONComments(data []byte) []byte {
	var result []byte
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		// Handle escape sequences in strings
		if inString && escaped {
			result = append(result, c)
			escaped = false
			continue
		}

		if inString && c == '\\' {
			result = append(result, c)
			escaped = true
			continue
		}

		// Track string boundaries
		if c == '"' && !inLineComment && !inBlockComment {
			inString = !inString
			result = append(result, c)
			continue
		}

		// Inside string - keep everything
		if inString {
			result = append(result, c)
			continue
		}

		// End of line comment
		if inLineComment {
			if c == '\n' {
				inLineComment = false
				result = append(result, c) // Keep newline
			}
			continue
		}

		// End of block comment
		if inBlockComment {
			if c == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i++ // Skip the '/'
			}
			continue
		}

		// Start of line comment
		if c == '/' && i+1 < len(data) && data[i+1] == '/' {
			inLineComment = true
			i++ // Skip second '/'
			continue
		}

		// Start of block comment
		if c == '/' && i+1 < len(data) && data[i+1] == '*' {
			inBlockComment = true
			i++ // Skip the '*'
			continue
		}

		// Regular character
		result = append(result, c)
	}

	return result
}

// stripTrailingCommas removes trailing commas before ] and } in JSON
func stripTrailingCommas(data []byte) []byte {
	var result []byte
	inString := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		// Handle escape sequences in strings
		if inString && escaped {
			result = append(result, c)
			escaped = false
			continue
		}

		if inString && c == '\\' {
			result = append(result, c)
			escaped = true
			continue
		}

		// Track string boundaries
		if c == '"' {
			inString = !inString
			result = append(result, c)
			continue
		}

		// Inside string - keep everything
		if inString {
			result = append(result, c)
			continue
		}

		// Check for trailing comma: comma followed by whitespace then ] or }
		if c == ',' {
			// Look ahead for ] or } (skipping whitespace)
			j := i + 1
			for j < len(data) && (data[j] == ' ' || data[j] == '\t' || data[j] == '\n' || data[j] == '\r') {
				j++
			}
			if j < len(data) && (data[j] == ']' || data[j] == '}') {
				// Skip the trailing comma
				continue
			}
		}

		result = append(result, c)
	}

	return result
}
