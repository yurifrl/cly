package jsonc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON map[string]interface{}
	}{
		{
			name:  "no comments",
			input: `{"key": "value"}`,
			wantJSON: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "single line comment",
			input: "{\n  // this is a comment\n  \"key\": \"value\"\n}",
			wantJSON: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "inline comment",
			input: "{\n  \"key\": \"value\" // inline\n}",
			wantJSON: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "block comment",
			input: "{\n  /* block comment */\n  \"key\": \"value\"\n}",
			wantJSON: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "multiline block comment",
			input: "{\n  /* multi\n     line */\n  \"key\": \"value\"\n}",
			wantJSON: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "comment inside string preserved",
			input: `{"key": "http://example.com"}`,
			wantJSON: map[string]interface{}{
				"key": "http://example.com",
			},
		},
		{
			name:  "trailing commas removed",
			input: "{\n  \"a\": 1,\n  \"b\": 2,\n}",
			wantJSON: map[string]interface{}{
				"a": float64(1),
				"b": float64(2),
			},
		},
		{
			name:  "trailing comma in array",
			input: `{"arr": [1, 2, 3,]}`,
			wantJSON: map[string]interface{}{
				"arr": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Strip([]byte(tt.input))
			require.NoError(t, err)
			assert.True(t, json.Valid(got))

			var m map[string]interface{}
			require.NoError(t, json.Unmarshal(got, &m))
			for k, wantV := range tt.wantJSON {
				assert.Equal(t, wantV, m[k])
			}
		})
	}
}

func TestStrip_Invalid(t *testing.T) {
	_, err := Strip([]byte(`{invalid`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not valid JSON")
}

func TestConvert_ExpandsEnv(t *testing.T) {
	t.Setenv("CLY_JSONC_HOME", "/home/test")
	t.Setenv("CLY_JSONC_EDITOR", "vim")

	tests := []struct {
		name  string
		input string
		key   string
		want  string
	}{
		{
			name:  "single ${VAR}",
			input: `{"home": "${CLY_JSONC_HOME}"}`,
			key:   "home",
			want:  "/home/test",
		},
		{
			name:  "bare $VAR",
			input: `{"editor": "$CLY_JSONC_EDITOR"}`,
			key:   "editor",
			want:  "vim",
		},
		{
			name:  "embedded in longer string",
			input: `{"path": "${CLY_JSONC_HOME}/bin/tool"}`,
			key:   "path",
			want:  "/home/test/bin/tool",
		},
		{
			name:  "unset becomes empty",
			input: `{"val": "${CLY_JSONC_UNSET_XXXX}"}`,
			key:   "val",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert([]byte(tt.input))
			require.NoError(t, err)
			var m map[string]interface{}
			require.NoError(t, json.Unmarshal(got, &m))
			assert.Equal(t, tt.want, m[tt.key])
		})
	}
}

func TestConvert_RespectsNoInterpolation(t *testing.T) {
	t.Setenv("CLY_JSONC_HOME", "/home/test")

	input := "// @no-interpolation\n{\n  \"home\": \"${CLY_JSONC_HOME}\"\n}"
	got, err := Convert([]byte(input))
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(got, &m))
	assert.Equal(t, "${CLY_JSONC_HOME}", m["home"])
}

func TestConvert_InvalidAfterExpansion(t *testing.T) {
	t.Setenv("CLY_JSONC_BAD", `has "quote" inside`)

	input := `{"key": "${CLY_JSONC_BAD}"}`
	_, err := Convert([]byte(input))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expansion produced invalid JSON")
}

func TestHasNoInterpolation(t *testing.T) {
	assert.True(t, HasNoInterpolation([]byte("// @no-interpolation\n{}\n")))
	assert.True(t, HasNoInterpolation([]byte("{\n// @no-interpolation\n}\n")))
	assert.False(t, HasNoInterpolation([]byte("{\"key\": \"value\"}\n")))

	// Past line 10 should not count.
	lines := ""
	for i := 0; i < 12; i++ {
		lines += "// line\n"
	}
	lines += "// @no-interpolation\n"
	assert.False(t, HasNoInterpolation([]byte(lines)))
}
