// sety.go parses --sety / --sety-string / --dry-run / --helpy flags
// out of the args slice before forwarding the rest to pi. The piwrap
// cobra command runs with DisableFlagParsing: true, so cobra never
// sees these — we hand-roll the scanner here.
package piwrap

import (
	"strconv"
	"strings"
)

// SetyKey enumerates the fixed key set accepted by --sety. Anything
// outside this list yields SETY_UNKNOWN_KEY.
type SetyKey string

const (
	SetyKeyImportID       SetyKey = "session_import.id"
	SetyKeyImportOverride SetyKey = "session_import.override"
)

// validSetyKeys lists every accepted --sety key. Used for validation
// and error hints.
var validSetyKeys = []string{
	string(SetyKeyImportID),
	string(SetyKeyImportOverride),
}

// SetyValues holds the parsed --sety key/value pairs in typed form.
// Unset fields are zero values; presence is tracked via the Has*
// booleans.
type SetyValues struct {
	ImportID         string
	HasImportID      bool
	ImportOverride   bool
	HasImportOverride bool
}

// piwrapFlags is the result of stripping piwrap-owned flags from
// args. Rest contains everything that should be forwarded to pi.
type piwrapFlags struct {
	Sety   SetyValues
	DryRun bool
	Helpy  bool
	// HelpyJSON is set when --helpy is followed by `-o json` or
	// `--output json`. The output flag itself is also stripped from
	// Rest so it doesn't reach pi.
	HelpyJSON bool
	Rest      []string
}

// extractPiwrapFlags scans args for --sety / --sety-string / --dry-run
// / --helpy and returns the parsed values plus the remaining args
// (forwarded to pi). Returns a *SetyError on bad input.
func extractPiwrapFlags(args []string) (piwrapFlags, *SetyError) {
	out := piwrapFlags{Rest: make([]string, 0, len(args))}

	for i := 0; i < len(args); i++ {
		a := args[i]

		switch {
		case a == "--sety" || a == "-y":
			if i+1 >= len(args) {
				return out, newSetyError(CodeSetyFormat,
					"--sety requires a key=value argument",
					map[string]interface{}{"flag": a})
			}
			if err := applySetyPair(&out.Sety, args[i+1], true); err != nil {
				return out, err
			}
			i++
		case strings.HasPrefix(a, "--sety="):
			if err := applySetyPair(&out.Sety, a[len("--sety="):], true); err != nil {
				return out, err
			}
		case strings.HasPrefix(a, "-y="):
			if err := applySetyPair(&out.Sety, a[len("-y="):], true); err != nil {
				return out, err
			}
		case a == "--sety-string":
			if i+1 >= len(args) {
				return out, newSetyError(CodeSetyFormat,
					"--sety-string requires a key=value argument", nil)
			}
			if err := applySetyPair(&out.Sety, args[i+1], false); err != nil {
				return out, err
			}
			i++
		case strings.HasPrefix(a, "--sety-string="):
			if err := applySetyPair(&out.Sety, a[len("--sety-string="):], false); err != nil {
				return out, err
			}
		case a == "--dry-run":
			out.DryRun = true
		case a == "--helpy":
			out.Helpy = true
		case a == "-o" || a == "--output":
			// Only consumed when --helpy is in effect. We can't know
			// at this point whether --helpy precedes or follows; do a
			// post-pass at the end. For now, capture and leave a
			// placeholder marker we'll handle below.
			if i+1 < len(args) {
				if out.Helpy && (args[i+1] == "json" || args[i+1] == "JSON") {
					out.HelpyJSON = true
					i++
					continue
				}
			}
			out.Rest = append(out.Rest, a)
		case strings.HasPrefix(a, "--output="):
			if out.Helpy && (a[len("--output="):] == "json" || a[len("--output="):] == "JSON") {
				out.HelpyJSON = true
				continue
			}
			out.Rest = append(out.Rest, a)
		default:
			out.Rest = append(out.Rest, a)
		}
	}

	return out, nil
}

// applySetyPair parses a single key=value pair and writes it into v.
// coerce=true enables boolean coercion for boolean keys; coerce=false
// (the --sety-string variant) keeps the value as-is.
func applySetyPair(v *SetyValues, kv string, coerce bool) *SetyError {
	eq := strings.IndexByte(kv, '=')
	if eq <= 0 {
		return newSetyError(CodeSetyFormat,
			"--sety expects key=value, got "+strconv.Quote(kv),
			map[string]interface{}{"got": kv})
	}
	key := kv[:eq]
	val := kv[eq+1:]

	switch SetyKey(key) {
	case SetyKeyImportID:
		v.ImportID = val
		v.HasImportID = true
		return nil
	case SetyKeyImportOverride:
		if coerce {
			b, err := parseBool(val)
			if err != nil {
				e := newSetyError(CodeSetyParse,
					"--sety session_import.override: cannot parse "+strconv.Quote(val)+" as bool",
					map[string]interface{}{"value": val})
				e.Hint = "use true|false|1|0, or pass --sety-string to keep as a string"
				return e
			}
			v.ImportOverride = b
			v.HasImportOverride = true
			return nil
		}
		// --sety-string for a bool is unusual; reject so users don't
		// silently disable coercion for a boolean key.
		return newSetyError(CodeSetyParse,
			"--sety-string is not supported for session_import.override (bool)", nil)
	default:
		e := newSetyError(CodeSetyUnknownKey,
			"unknown --sety key: "+strconv.Quote(key),
			map[string]interface{}{"key": key, "valid": validSetyKeys})
		e.Hint = "valid keys: " + strings.Join(validSetyKeys, ", ")
		return e
	}
}

// parseBool accepts true/false/1/0 (case-insensitive) and rejects
// everything else. strconv.ParseBool is too lenient (accepts t/T/yes
// inconsistently across versions), so we do it ourselves.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, errInvalidBool
	}
}

// sentinel; we never inspect the underlying error message, only the
// fact that parseBool returned non-nil.
var errInvalidBool = errInvalidBoolErr{}

type errInvalidBoolErr struct{}

func (errInvalidBoolErr) Error() string { return "invalid boolean" }
