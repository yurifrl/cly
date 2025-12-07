package builder

import (
	"strings"
	"unicode"
)

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	// Remove leading/trailing whitespace
	s = strings.TrimSpace(s)

	// Split words manually to handle camelCase boundaries
	words := splitWords(s)

	if len(words) == 0 {
		return ""
	}

	// Convert first word to lowercase
	result := strings.ToLower(words[0])

	// Convert subsequent words to title case (first letter uppercase, rest lowercase)
	for i := 1; i < len(words); i++ {
		word := words[i]
		if len(word) > 0 {
			// Convert to title case
			titleWord := strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			result += titleWord
		}
	}

	return result
}

// splitWords splits a string into words, handling various separators and camelCase boundaries
func splitWords(s string) []string {
	if s == "" {
		return nil
	}

	var words []string
	var currentWord strings.Builder

	runes := []rune(s)

	for i, r := range runes {
		// Check if this is a separator character
		if r == ' ' || r == '_' || r == '-' || r == '\t' || r == '\n' {
			// End current word if it exists
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			continue
		}

		// Check for camelCase boundary (lowercase followed by uppercase)
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			// End current word and start new one
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}

		// Check for boundary between consecutive uppercase letters followed by lowercase
		// e.g., "XMLHttpRequest" should split into "XML", "Http", "Request"
		if i > 0 && i < len(runes)-1 && unicode.IsUpper(runes[i-1]) && unicode.IsUpper(r) && unicode.IsLower(runes[i+1]) {
			// End current word and start new one
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}

		currentWord.WriteRune(r)
	}

	// Add final word
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// ToCamelCaseSimple is a simpler version that only handles space, underscore, and hyphen separators
func ToCamelCaseSimple(s string) string {
	if s == "" {
		return s
	}

	// Replace separators with spaces and split
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)

	if len(words) == 0 {
		return ""
	}

	// Convert first word to lowercase
	result := strings.ToLower(words[0])

	// Convert subsequent words to title case
	for i := 1; i < len(words); i++ {
		word := words[i]
		if len(word) > 0 {
			// Capitalize first letter, lowercase the rest
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			for j := 1; j < len(runes); j++ {
				runes[j] = unicode.ToLower(runes[j])
			}
			result += string(runes)
		}
	}

	return result
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	// Remove leading/trailing whitespace
	s = strings.TrimSpace(s)

	// Split words manually to handle camelCase boundaries
	words := splitWords(s)

	// Filter out empty strings and convert to lowercase
	var filteredWords []string
	for _, word := range words {
		if word != "" {
			filteredWords = append(filteredWords, strings.ToLower(word))
		}
	}

	if len(filteredWords) == 0 {
		return ""
	}

	// Join words with underscores
	return strings.Join(filteredWords, "_")
}

// ToKebabCase converts a string to kebab-case
func ToKebabCase(s string) string {
	if s == "" {
		return s
	}

	// Remove leading/trailing whitespace
	s = strings.TrimSpace(s)

	// Split words manually to handle camelCase boundaries
	words := splitWords(s)

	// Filter out empty strings and convert to lowercase
	var filteredWords []string
	for _, word := range words {
		if word != "" {
			filteredWords = append(filteredWords, strings.ToLower(word))
		}
	}

	if len(filteredWords) == 0 {
		return ""
	}

	// Join words with hyphens
	return strings.Join(filteredWords, "-")
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Remove leading/trailing whitespace
	s = strings.TrimSpace(s)

	// Split words manually to handle camelCase boundaries
	words := splitWords(s)

	if len(words) == 0 {
		return ""
	}

	// Convert all words to title case
	for i := 0; i < len(words); i++ {
		if len(words[i]) > 0 {
			runes := []rune(words[i])
			runes[0] = unicode.ToUpper(runes[0])
			for j := 1; j < len(runes); j++ {
				runes[j] = unicode.ToLower(runes[j])
			}
			words[i] = string(runes)
		}
	}

	return strings.Join(words, "")
}
