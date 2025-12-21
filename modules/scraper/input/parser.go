package input

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing product IDs from various input formats
type Parser struct{}

// NewParser creates a new input parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseURLs parses a comma-separated string of URLs or IDs
func (p *Parser) ParseURLs(input string) ([]string, error) {
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Split by comma
	parts := strings.Split(input, ",")
	var ids []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id := ExtractProductID(part)
		if id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid product IDs found")
	}

	return ids, nil
}

// ParseFile parses a file containing product IDs/URLs
func (p *Parser) ParseFile(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine file type by extension
	ext := strings.ToLower(filePath)
	if strings.HasSuffix(ext, ".json") {
		return p.ParseJSON(data)
	} else if strings.HasSuffix(ext, ".yaml") || strings.HasSuffix(ext, ".yml") {
		return p.ParseYAML(data)
	} else if strings.HasSuffix(ext, ".csv") {
		return p.ParseCSV(string(data))
	}

	// Default: treat as newline-separated text
	return p.ParseText(string(data))
}

// ParseText parses newline or comma-separated text
func (p *Parser) ParseText(content string) ([]string, error) {
	// Try comma-separated first
	if strings.Contains(content, ",") {
		return p.ParseURLs(content)
	}

	// Newline-separated
	lines := strings.Split(content, "\n")
	var ids []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		id := ExtractProductID(line)
		if id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid product IDs found")
	}

	return ids, nil
}

// ParseCSV parses CSV content
func (p *Parser) ParseCSV(content string) ([]string, error) {
	r := csv.NewReader(strings.NewReader(content))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	var ids []string

	// Check if first row is header
	firstRow := records[0]
	hasHeader := false
	idColumn := 0

	// Look for product_id or id column
	for i, col := range firstRow {
		col = strings.ToLower(strings.TrimSpace(col))
		if col == "product_id" || col == "id" || col == "productid" {
			hasHeader = true
			idColumn = i
			break
		}
	}

	// Parse data rows
	startRow := 0
	if hasHeader {
		startRow = 1
	}

	for _, record := range records[startRow:] {
		if len(record) > idColumn {
			id := ExtractProductID(record[idColumn])
			if id != "" {
				ids = append(ids, id)
			}
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid product IDs found in CSV")
	}

	return ids, nil
}

// ParseJSON parses JSON content
func (p *Parser) ParseJSON(content []byte) ([]string, error) {
	// Try parsing as string array
	var strArray []string
	if err := json.Unmarshal(content, &strArray); err == nil {
		var ids []string
		for _, item := range strArray {
			id := ExtractProductID(item)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			return ids, nil
		}
	}

	// Try parsing as array of objects
	var objArray []map[string]interface{}
	if err := json.Unmarshal(content, &objArray); err == nil {
		var ids []string
		for _, obj := range objArray {
			// Look for url, id, product_id, or productId fields
			for _, key := range []string{"url", "id", "product_id", "productId"} {
				if val, ok := obj[key]; ok {
					if str, ok := val.(string); ok {
						id := ExtractProductID(str)
						if id != "" {
							ids = append(ids, id)
							break
						}
					}
				}
			}
		}
		if len(ids) > 0 {
			return ids, nil
		}
	}

	return nil, fmt.Errorf("invalid JSON format: expected array of strings or objects")
}

// ParseYAML parses YAML content
func (p *Parser) ParseYAML(content []byte) ([]string, error) {
	// Try parsing as string array
	var strArray []string
	if err := yaml.Unmarshal(content, &strArray); err == nil {
		var ids []string
		for _, item := range strArray {
			id := ExtractProductID(item)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			return ids, nil
		}
	}

	// Try parsing as array of maps
	var objArray []map[string]interface{}
	if err := yaml.Unmarshal(content, &objArray); err == nil {
		var ids []string
		for _, obj := range objArray {
			for _, key := range []string{"url", "id", "product_id", "productId"} {
				if val, ok := obj[key]; ok {
					if str, ok := val.(string); ok {
						id := ExtractProductID(str)
						if id != "" {
							ids = append(ids, id)
							break
						}
					}
				}
			}
		}
		if len(ids) > 0 {
			return ids, nil
		}
	}

	return nil, fmt.Errorf("invalid YAML format: expected array of strings or objects")
}

// ExtractProductID extracts product ID from URL or returns the ID if already clean
func ExtractProductID(urlOrID string) string {
	urlOrID = strings.TrimSpace(urlOrID)
	if urlOrID == "" {
		return ""
	}

	// If it's already just digits, return it
	if regexp.MustCompile(`^\d+$`).MatchString(urlOrID) {
		return urlOrID
	}

	// Extract from AliExpress URL pattern: /item/1005003618976317.html
	re := regexp.MustCompile(`/item/(\d+)\.html`)
	matches := re.FindStringSubmatch(urlOrID)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find any sequence of digits
	re = regexp.MustCompile(`(\d{10,})`) // AliExpress IDs are typically 13+ digits
	matches = re.FindStringSubmatch(urlOrID)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
