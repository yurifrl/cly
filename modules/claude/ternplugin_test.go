package claude

import (
	"testing"
	"testing/fstest"
)

func TestEmbeddedTernPluginHasVersion(t *testing.T) {
	v, err := ternPluginVersion(ternPluginFS, "tern-plugin/plugin.toml")
	if err != nil || v == "" {
		t.Fatalf("embedded plugin.toml version: %q, %v", v, err)
	}
}

func TestTernPluginVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"ok.toml":   {Data: []byte("schema = 1\nid = \"x\"\nversion = \"1.2.3\"\n")},
		"none.toml": {Data: []byte("schema = 1\n")},
	}
	if v, err := ternPluginVersion(fsys, "ok.toml"); err != nil || v != "1.2.3" {
		t.Fatalf("got %q, %v", v, err)
	}
	if _, err := ternPluginVersion(fsys, "none.toml"); err == nil {
		t.Fatal("want error for missing version")
	}
	if _, err := ternPluginVersion(fsys, "missing.toml"); err == nil {
		t.Fatal("want error for missing file")
	}
}
