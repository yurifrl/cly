// render_json.go produces the machine-readable form of the cheat
// sheet. Doubles as the source for `cly pi schema` / --describe.
package helpy

import (
	"encoding/json"
	"io"
)

// JSONOutput is the top-level shape returned by RenderJSON.
type JSONOutput struct {
	Version  string              `json:"version,omitempty"`
	Sections []JSONSection       `json:"sections"`
	SeeAlso  map[string]string   `json:"see_also,omitempty"`
}

// JSONSection groups entries by their Section name.
type JSONSection struct {
	Title   string  `json:"title"`
	Entries []Entry `json:"flags"`
}

// RenderJSON writes the registry to w as a single JSON document.
// version is rendered as-is; pass empty string to omit. seeAlso is
// optional; nil omits the field.
func RenderJSON(w io.Writer, version string, seeAlso map[string]string) error {
	all := All()

	sections := []JSONSection{}
	idx := map[string]int{}
	for _, e := range all {
		i, ok := idx[e.Section]
		if !ok {
			sections = append(sections, JSONSection{Title: e.Section})
			i = len(sections) - 1
			idx[e.Section] = i
		}
		sections[i].Entries = append(sections[i].Entries, e)
	}

	out := JSONOutput{
		Version:  version,
		Sections: sections,
		SeeAlso:  seeAlso,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}
