// render_text.go produces the lipgloss-styled cheat sheet. Strips
// styling when stdout is not a TTY or NO_COLOR is set.
package helpy

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// RenderText writes a human-readable cheat sheet to w. Header is
// shown above all sections; trailer is shown below.
func RenderText(w io.Writer, header, trailer string) {
	if header != "" {
		fmt.Fprintln(w, header)
		fmt.Fprintln(w)
	}

	currentSection := ""
	for _, e := range All() {
		if e.Section != currentSection {
			currentSection = e.Section
			fmt.Fprintln(w, strings.ToUpper(e.Section))
		}
		writeEntry(w, e)
	}

	if trailer != "" {
		fmt.Fprintln(w)
		fmt.Fprint(w, trailer)
	}
}

func writeEntry(w io.Writer, e Entry) {
	flagLine := strings.Join(e.Flags, ", ")
	if e.Value != "" {
		flagLine += " " + e.Value
	}
	fmt.Fprintln(w, "  "+flagLine)

	for _, line := range strings.Split(strings.TrimRight(e.Description, "\n"), "\n") {
		fmt.Fprintln(w, "      "+line)
	}

	if len(e.Requires) > 0 {
		fmt.Fprintln(w, "      Requires: "+strings.Join(e.Requires, ", "))
	}
	if len(e.ConfigKeys) > 0 {
		fmt.Fprintln(w, "      Config:   "+strings.Join(e.ConfigKeys, ", "))
	}
	if len(e.EnvVars) > 0 {
		fmt.Fprintln(w, "      Env:      "+strings.Join(e.EnvVars, ", "))
	}
	if len(e.Errors) > 0 {
		fmt.Fprintln(w, "      Errors:   "+strings.Join(e.Errors, ", "))
	}
	if len(e.Examples) > 0 {
		fmt.Fprintln(w, "      Examples:")
		for _, ex := range e.Examples {
			fmt.Fprintln(w, "        "+ex)
		}
	}
	fmt.Fprintln(w)
}

// useColor reports whether terminal styling should be emitted.
// Reserved for future lipgloss integration; currently we ship plain
// text only because the cheat sheet readability is the priority.
func useColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
