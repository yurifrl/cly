// errors.go defines the SETY_* error codes used by piwrap. Each Err
// carries a stable Code that's safe to render as JSON on stdout, and
// a human-readable Message. Hint is rendered on stderr only and
// never enters the JSON payload.
package piwrap

import (
	"encoding/json"
	"fmt"
	"os"
)

// SetyError is a structured error returned by piwrap's pre-pi pipeline.
// It implements error so it can flow through Run() and be rendered by
// the caller.
type SetyError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Hint    string                 `json:"-"`
}

func (e *SetyError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// newSetyError constructs a SetyError. details may be nil.
func newSetyError(code, msg string, details map[string]interface{}) *SetyError {
	return &SetyError{Code: code, Message: msg, Details: details}
}

// Render writes the structured error as JSON to stdout, the hint (if
// any) to stderr, and returns a process-friendly error.
func (e *SetyError) Render() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]interface{}{
		"error":   "piwrap",
		"code":    e.Code,
		"message": e.Message,
		"details": e.Details,
	})
	if e.Hint != "" {
		fmt.Fprintln(os.Stderr, "Hint:", e.Hint)
	}
}

// Stable error codes. Keep in sync with the design doc and --helpy.
const (
	CodeSetyFormat            = "SETY_FORMAT"
	CodeSetyUnknownKey        = "SETY_UNKNOWN_KEY"
	CodeSetyParse             = "SETY_PARSE"
	CodeSetyNameRequired      = "SETY_NAME_REQUIRED"
	CodeSetyImportIDTooShort  = "SETY_IMPORT_ID_TOO_SHORT"
	CodeSetyImportNotFound    = "SETY_IMPORT_NOT_FOUND"
	CodeSetyImportAmbiguous   = "SETY_IMPORT_AMBIGUOUS"
	CodeSetyImportConflict    = "SETY_IMPORT_CONFLICT"
	CodeSetyImportTargetBusy  = "SETY_IMPORT_TARGET_BUSY"
	CodeSetyImportFailed      = "SETY_IMPORT_FAILED"
)
