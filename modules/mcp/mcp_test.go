package mcp

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var update = flag.Bool("update", false, "update golden files")

// ========== INTERACTION TESTS ==========

func TestUpdate_CursorMovement(t *testing.T) {
	tests := []struct {
		name       string
		key        tea.KeyMsg
		wantCursor int
		wantScroll int
	}{
		{
			name:       "move down with j",
			key:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
			wantCursor: 1,
			wantScroll: 0,
		},
		{
			name:       "move down with arrow",
			key:        tea.KeyMsg{Type: tea.KeyDown},
			wantCursor: 1,
			wantScroll: 0,
		},
		{
			name:       "move up with k",
			key:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")},
			wantCursor: 0,
			wantScroll: 0,
		},
		{
			name:       "move up with arrow",
			key:        tea.KeyMsg{Type: tea.KeyUp},
			wantCursor: 0,
			wantScroll: 0,
		},
		{
			name:       "home key",
			key:        tea.KeyMsg{Type: tea.KeyHome},
			wantCursor: 0,
			wantScroll: 0,
		},
		{
			name:       "end key",
			key:        tea.KeyMsg{Type: tea.KeyEnd},
			wantCursor: 2, // Last item
			wantScroll: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := createTestModel()
			if tt.name == "move up with k" || tt.name == "move up with arrow" {
				testModel.cursor = 1
			}

			updatedModel, _ := testModel.Update(tt.key)
			m := updatedModel.(Model)

			if m.cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.wantCursor)
			}
			if m.scrollOffset != tt.wantScroll {
				t.Errorf("scrollOffset = %d, want %d", m.scrollOffset, tt.wantScroll)
			}
		})
	}
}

func TestUpdate_ExpandCollapse(t *testing.T) {
	model := createTestModelWithPresets()
	model.cursor = 0

	// Test expand with right arrow
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	m := updatedModel.(Model)

	if !m.expandedPresets["webdev"] {
		t.Error("preset should be expanded after right arrow")
	}

	// Test collapse with left arrow
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updatedModel.(Model)

	if m.expandedPresets["webdev"] {
		t.Error("preset should be collapsed after left arrow")
	}

	// Test expand with 'l' key
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updatedModel.(Model)

	if !m.expandedPresets["webdev"] {
		t.Error("preset should be expanded after 'l' key")
	}

	// Test collapse with 'h' key
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = updatedModel.(Model)

	if m.expandedPresets["webdev"] {
		t.Error("preset should be collapsed after 'h' key")
	}
}

func TestUpdate_SpaceToggle(t *testing.T) {
	model := createTestModel()
	model.cursor = 0
	model.checkedMCPs["github"] = false

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	m := updatedModel.(Model)

	if !m.checkedMCPs["github"] {
		t.Error("MCP should be checked after space")
	}

	// Toggle again
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updatedModel.(Model)

	if m.checkedMCPs["github"] {
		t.Error("MCP should be unchecked after second space")
	}
}

func TestUpdate_SearchMode(t *testing.T) {
	model := createTestModel()

	// Enter search mode with '/'
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m := updatedModel.(Model)

	if !m.searchFocused {
		t.Error("should be in search mode after '/'")
	}

	// Exit search with ESC
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedModel.(Model)

	if m.searchFocused {
		t.Error("should exit search mode after ESC")
	}
}

func TestUpdate_HelpToggle(t *testing.T) {
	model := createTestModel()

	// Toggle help with '?'
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m := updatedModel.(Model)

	if !m.showHelp {
		t.Error("help should be shown after '?'")
	}

	// Navigate in help with arrows
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updatedModel.(Model)

	if m.helpScrollOffset != 1 {
		t.Error("help should scroll down")
	}

	// Close help with ESC
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedModel.(Model)

	if m.showHelp {
		t.Error("help should be closed after ESC")
	}
}

// ========== CATALOG TESTS ==========

func TestCatalog_LoadFromYAML(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()
	mcpsDir := filepath.Join(tmpDir, "mcps")
	os.MkdirAll(mcpsDir, 0755)

	// Create test YAML file
	yamlContent := `github:
  command: npx
  args:
    - "-y"
    - "@modelcontextprotocol/server-github"
  tags:
    - vcs
    - git
filesystem:
  command: npx
  args:
    - "-y"
    - "@modelcontextprotocol/server-filesystem"
    - "/tmp"
`
	err := os.WriteFile(filepath.Join(mcpsDir, "test.yaml"), []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	catalog, err := LoadCatalog(tmpDir)
	if err != nil {
		t.Fatalf("LoadCatalog failed: %v", err)
	}

	mcps := catalog.GetAll()
	if len(mcps) != 2 {
		t.Errorf("expected 2 MCPs, got %d", len(mcps))
	}

	github, ok := catalog.Get("github")
	if !ok {
		t.Error("github MCP not found")
	}
	if github.Command != "npx" {
		t.Errorf("github command = %q, want 'npx'", github.Command)
	}
	if len(github.Tags) != 2 {
		t.Errorf("github tags count = %d, want 2", len(github.Tags))
	}
}

func TestCatalog_LoadFromJSONC(t *testing.T) {
	tmpDir := t.TempDir()
	mcpsDir := filepath.Join(tmpDir, "mcps")
	os.MkdirAll(mcpsDir, 0755)

	// Create test JSONC file with comments and URLs
	jsoncContent := `{
  // This is a comment
  "github": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "tags": ["vcs"]
  },
  "remote": {
    "command": "npx",
    "args": ["mcp-remote", "https://example.com/mcp"]
  }
}
`
	err := os.WriteFile(filepath.Join(mcpsDir, "test.jsonc"), []byte(jsoncContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	catalog, err := LoadCatalog(tmpDir)
	if err != nil {
		t.Fatalf("LoadCatalog failed: %v", err)
	}

	mcps := catalog.GetAll()
	if len(mcps) != 2 {
		t.Errorf("expected 2 MCPs, got %d", len(mcps))
	}

	remote, ok := catalog.Get("remote")
	if !ok {
		t.Error("remote MCP not found")
	}
	// Verify URL in args wasn't corrupted by comment stripping
	if len(remote.Args) != 2 || remote.Args[1] != "https://example.com/mcp" {
		t.Errorf("remote args = %v, URL should be preserved", remote.Args)
	}
}

func TestCatalog_Filter(t *testing.T) {
	tmpDir := t.TempDir()
	mcpsDir := filepath.Join(tmpDir, "mcps")
	os.MkdirAll(mcpsDir, 0755)

	yamlContent := `github:
  command: npx
  tags: [vcs, git]
  description: GitHub integration
filesystem:
  command: npx
  tags: [file]
kubernetes:
  command: kubectl
  tags: [k8s, cloud]
`
	os.WriteFile(filepath.Join(mcpsDir, "test.yaml"), []byte(yamlContent), 0644)

	catalog, _ := LoadCatalog(tmpDir)

	// Filter by query
	results := catalog.Filter("git", nil)
	if len(results) != 1 || results[0].Name != "github" {
		t.Errorf("filter by 'git' should return github, got %v", results)
	}

	// Filter by tag
	results = catalog.Filter("", []string{"vcs"})
	if len(results) != 1 || results[0].Name != "github" {
		t.Errorf("filter by tag 'vcs' should return github, got %v", results)
	}

	// Filter by description
	results = catalog.Filter("integration", nil)
	if len(results) != 1 || results[0].Name != "github" {
		t.Errorf("filter by 'integration' should return github, got %v", results)
	}
}

// ========== VALIDATION TESTS ==========

func TestValidation_MissingCommand(t *testing.T) {
	mcp := MCP{Name: "broken", Command: "", URL: ""}
	issues := validateMCP(mcp)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != SeverityError {
		t.Error("missing command should be an error")
	}
}

func TestValidation_CommandNotInPath(t *testing.T) {
	mcp := MCP{Name: "test", Command: "nonexistent-command-xyz"}
	issues := validateMCP(mcp)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != SeverityWarning {
		t.Error("command not in PATH should be a warning")
	}
}

func TestValidation_ValidMCP(t *testing.T) {
	mcp := MCP{Name: "test", Command: "echo", Args: []string{"hello"}}
	issues := validateMCP(mcp)

	if len(issues) != 0 {
		t.Errorf("valid MCP should have no issues, got %v", issues)
	}
}

// ========== RENDERER TESTS (GOLDEN FILES) ==========

func TestView_BasicRender(t *testing.T) {
	lipgloss.SetColorProfile(0)

	model := createTestModel()
	output := model.View()
	output = stripAnsiCodes(output)

	compareWithGolden(t, output, "view_basic.golden")
}

func TestView_WithCursor(t *testing.T) {
	lipgloss.SetColorProfile(0)

	model := createTestModel()
	model.cursor = 1

	output := model.View()
	output = stripAnsiCodes(output)

	compareWithGolden(t, output, "view_cursor.golden")
}

// ========== HELPER FUNCTIONS ==========

func createTestModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Search MCPs..."

	return Model{
		catalog: nil,
		availableMCPs: []MCP{
			{Name: "github", Tags: []string{"vcs", "git"}},
			{Name: "filesystem", Tags: []string{"file", "io"}},
			{Name: "kubernetes", Tags: []string{"k8s", "cloud"}},
		},
		displayItems: []ListItem{
			{Type: ListItemMCP, Name: "github", MCP: &MCP{Name: "github", Tags: []string{"vcs", "git"}}},
			{Type: ListItemMCP, Name: "filesystem", MCP: &MCP{Name: "filesystem", Tags: []string{"file", "io"}}},
			{Type: ListItemMCP, Name: "kubernetes", MCP: &MCP{Name: "kubernetes", Tags: []string{"k8s", "cloud"}}},
		},
		filteredItems: []ListItem{
			{Type: ListItemMCP, Name: "github", MCP: &MCP{Name: "github", Tags: []string{"vcs", "git"}}},
			{Type: ListItemMCP, Name: "filesystem", MCP: &MCP{Name: "filesystem", Tags: []string{"file", "io"}}},
			{Type: ListItemMCP, Name: "kubernetes", MCP: &MCP{Name: "kubernetes", Tags: []string{"k8s", "cloud"}}},
		},
		installedMCPs:   map[string]bool{"github": true},
		checkedMCPs:     map[string]bool{"github": true, "filesystem": false, "kubernetes": false},
		expandedPresets: make(map[string]bool),
		expandedTags:    make(map[string]bool),
		hiddenSections:  make(map[string]bool),
		context:         Context{AI: "claude", Scope: "user"},
		contextSource:   "default",
		cursor:          0,
		viewportHeight:  20,
		searchInput:     ti,
	}
}

func createTestModelWithPresets() Model {
	model := createTestModel()

	// Create a catalog with the test MCPs
	catalog := &Catalog{
		mcps: map[string]MCP{
			"github":     {Name: "github", Tags: []string{"vcs", "git"}},
			"filesystem": {Name: "filesystem", Tags: []string{"file", "io"}},
			"kubernetes": {Name: "kubernetes", Tags: []string{"k8s", "cloud"}},
		},
	}
	model.catalog = catalog

	preset := ListItem{
		Type:     ListItemPreset,
		Name:     "webdev",
		MCPNames: []string{"github", "filesystem"},
	}

	model.displayItems = append([]ListItem{preset}, model.displayItems...)
	model.filteredItems = append([]ListItem{preset}, model.filteredItems...)
	model.expandedPresets = map[string]bool{"webdev": false}

	return model
}

func stripAnsiCodes(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func compareWithGolden(t *testing.T, output, filename string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", filename)

	if *update {
		os.MkdirAll("testdata", 0755)
		err := os.WriteFile(goldenPath, []byte(output), 0644)
		if err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Skipf("Golden file not found (run with -update to create): %s", goldenPath)
		return
	}

	expectedTrimmed := strings.TrimSpace(string(expected))
	outputTrimmed := strings.TrimSpace(output)

	if expectedTrimmed != outputTrimmed {
		t.Errorf("View output doesn't match golden file %s\nRun with -update to regenerate", filename)
	}
}
