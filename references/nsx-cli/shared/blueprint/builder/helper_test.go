package builder

import (
	"testing"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "helloWorld"},
		{"Hello World", "helloWorld"},
		{"hello_world", "helloWorld"},
		{"hello-world", "helloWorld"},
		{"Hello-World", "helloWorld"},
		{"  leading space", "leadingSpace"},
		{"trailing space  ", "trailingSpace"},
		{"  multiple   spaces  ", "multipleSpaces"},
	}

	for _, test := range tests {
		result := ToCamelCase(test.input)
		if result != test.expected {
			t.Errorf("ToCamelCase(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello_world"},
		{"Hello World", "hello_world"},
		{"helloWorld", "hello_world"},
		{"HelloWorld", "hello_world"},
		{"hello-world", "hello_world"},
		{"Hello-World", "hello_world"},
		{"  leading space", "leading_space"},
		{"trailing space  ", "trailing_space"},
	}

	for _, test := range tests {
		result := ToSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("ToSnakeCase(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello-world"},
		{"Hello World", "hello-world"},
		{"helloWorld", "hello-world"},
		{"HelloWorld", "hello-world"},
		{"hello_world", "hello-world"},
		{"Hello_World", "hello-world"},
		{"  leading space", "leading-space"},
		{"trailing space  ", "trailing-space"},
	}

	for _, test := range tests {
		result := ToKebabCase(test.input)
		if result != test.expected {
			t.Errorf("ToKebabCase(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "HelloWorld"},
		{"Hello World", "HelloWorld"},
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"  leading space", "LeadingSpace"},
		{"trailing space  ", "TrailingSpace"},
	}

	for _, test := range tests {
		result := ToPascalCase(test.input)
		if result != test.expected {
			t.Errorf("ToPascalCase(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
