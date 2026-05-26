// import.go resolves a session_import.id to a source pi session
// JSONL file, handles target conflicts (quarantine), and runs the
// fork pipeline (delegated to fork.go).
package piwrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// minIDPrefix is the minimum prefix length for partial UUID lookup.
// Below this, collisions become statistically likely on busy session
// directories.
const minIDPrefix = 8

// uuidRegex matches a full lowercase UUID (with dashes).
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// SearchScope controls where resolveSource looks.
type SearchScope string

const (
	ScopeCwd SearchScope = "cwd"
	ScopeAll SearchScope = "all"
)

// importPlan captures the resolved state of an import operation. Used
// for both the real run and --dry-run rendering.
type importPlan struct {
	Source           string `json:"source"`
	Target           string `json:"target"`
	Conflict         bool   `json:"conflict"`
	Override         bool   `json:"override"`
	WouldQuarantine  string `json:"would_quarantine,omitempty"`
	WouldFork        bool   `json:"would_fork"`
	BlockedBy        string `json:"blocked_by,omitempty"`
}

// resolveSource maps a user-supplied id (full UUID, >=8-char prefix,
// or absolute .jsonl path) to a single source path. Returns a
// *SetyError on any failure mode.
func resolveSource(id, sessionsRoot, cwdEncoded string, scope SearchScope) (string, *SetyError) {
	// Absolute path escape hatch.
	if filepath.IsAbs(id) {
		if !strings.HasSuffix(id, ".jsonl") {
			return "", newSetyError(CodeSetyImportFailed,
				"absolute --sety session_import.id must end in .jsonl",
				map[string]interface{}{"id": id})
		}
		if _, err := os.Stat(id); err != nil {
			return "", newSetyError(CodeSetyImportNotFound,
				"source path not found: "+id,
				map[string]interface{}{"id": id})
		}
		return id, nil
	}

	if len(id) < minIDPrefix {
		return "", newSetyError(CodeSetyImportIDTooShort,
			fmt.Sprintf("session_import.id must be >=%d chars (or an absolute .jsonl path)", minIDPrefix),
			map[string]interface{}{"id": id, "min": minIDPrefix})
	}

	roots := []string{}
	switch scope {
	case ScopeCwd:
		roots = append(roots, filepath.Join(sessionsRoot, cwdEncoded))
	case ScopeAll:
		entries, err := os.ReadDir(sessionsRoot)
		if err != nil {
			return "", newSetyError(CodeSetyImportFailed,
				"read sessions root: "+err.Error(), nil)
		}
		for _, e := range entries {
			if e.IsDir() {
				roots = append(roots, filepath.Join(sessionsRoot, e.Name()))
			}
		}
	default:
		return "", newSetyError(CodeSetyImportFailed,
			"invalid search_scope: "+string(scope), nil)
	}

	candidates := []string{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			// Missing per-cwd dir is normal — skip.
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			if strings.Contains(name, id) {
				candidates = append(candidates, filepath.Join(root, name))
			}
		}
	}

	switch len(candidates) {
	case 0:
		return "", newSetyError(CodeSetyImportNotFound,
			"no pi session matches id "+id,
			map[string]interface{}{"id": id, "scope": string(scope)})
	case 1:
		return candidates[0], nil
	default:
		e := newSetyError(CodeSetyImportAmbiguous,
			fmt.Sprintf("%d sessions match id %q", len(candidates), id),
			map[string]interface{}{"id": id, "candidates": candidates})
		e.Hint = "use a longer prefix or an absolute .jsonl path"
		return "", e
	}
}

// quarantineExisting moves an existing target file into the
// quarantine directory and returns the new path. Filename embeds a
// UTC timestamp + a sanitized version of the original target so
// restoration is unambiguous.
func quarantineExisting(target, quarantineDir string) (string, error) {
	if err := os.MkdirAll(quarantineDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir quarantine: %w", err)
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	// Encode target's path into the filename so the quarantined file
	// records where it came from. Replace separators with `-` and
	// drop leading separator (mirrors encodeCwd shape).
	encoded := strings.TrimPrefix(target, string(filepath.Separator))
	encoded = strings.ReplaceAll(encoded, string(filepath.Separator), "-")
	dst := filepath.Join(quarantineDir, ts+"-"+encoded)
	if err := os.Rename(target, dst); err != nil {
		return "", fmt.Errorf("move to quarantine: %w", err)
	}
	return dst, nil
}

// restoreFromQuarantine moves a quarantined file back to its
// original target path. Best-effort cleanup used when fork fails
// after we already quarantined the existing target.
func restoreFromQuarantine(quarantined, target string) error {
	if quarantined == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(quarantined, target)
}

// runImport orchestrates the full import pipeline: resolve, conflict
// check, quarantine if needed, fork. Returns the populated importPlan
// and any *SetyError. dryRun=true skips all filesystem mutations.
func runImport(values SetyValues, name, home, cwd string, dryRun bool) (importPlan, *SetyError) {
	cfg := loadImportConfig()
	plan := importPlan{Override: values.ImportOverride || (!values.HasImportOverride && cfg.Override)}

	if !values.HasImportOverride {
		plan.Override = cfg.Override
	}

	cwdEncoded := encodeCwd(cwd)
	sessionsRoot := filepath.Join(home, ".pi", "agent", "sessions")

	source, sErr := resolveSource(values.ImportID, sessionsRoot, cwdEncoded, cfg.SearchScope)
	if sErr != nil {
		return plan, sErr
	}
	plan.Source = source

	target := buildSessionPath(home, cwd, sessionFileNamePrefix(), name)
	plan.Target = target

	if source == target {
		// Self-import — no-op, signal to caller via WouldFork=false
		// and a non-error return. Caller will warn on stderr.
		plan.BlockedBy = "self_import"
		return plan, nil
	}

	if _, err := os.Stat(target); err == nil {
		plan.Conflict = true
		if !plan.Override {
			plan.BlockedBy = CodeSetyImportConflict
			if dryRun {
				return plan, nil
			}
			e := newSetyError(CodeSetyImportConflict,
				"target already exists: "+target,
				map[string]interface{}{"target": target, "source": source})
			e.Hint = "pass --sety session_import.override=true to move the existing file aside"
			return plan, e
		}
		// Override path: planned quarantine target.
		plan.WouldQuarantine = predictQuarantinePath(target, cfg.QuarantineDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return plan, newSetyError(CodeSetyImportFailed,
			"stat target: "+err.Error(),
			map[string]interface{}{"target": target})
	}

	plan.WouldFork = true
	if dryRun {
		return plan, nil
	}

	// Real run — quarantine if needed, then fork.
	var quarantined string
	if plan.Conflict {
		q, err := quarantineExisting(target, cfg.QuarantineDir)
		if err != nil {
			return plan, newSetyError(CodeSetyImportFailed,
				"quarantine existing target: "+err.Error(),
				map[string]interface{}{"target": target})
		}
		quarantined = q
		plan.WouldQuarantine = q
		fmt.Fprintf(os.Stderr, "moved existing session: %s -> %s\n", target, q)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		// Try to restore quarantined file before reporting.
		_ = restoreFromQuarantine(quarantined, target)
		return plan, newSetyError(CodeSetyImportFailed,
			"mkdir target dir: "+err.Error(), nil)
	}

	if err := forkSession(source, target); err != nil {
		_ = restoreFromQuarantine(quarantined, target)
		return plan, newSetyError(CodeSetyImportFailed,
			"fork session: "+err.Error(),
			map[string]interface{}{"source": source, "target": target})
	}

	return plan, nil
}

// predictQuarantinePath computes the path quarantineExisting would
// generate, used for dry-run reporting. Uses a fixed timestamp
// placeholder because the real timestamp is generated at execution.
func predictQuarantinePath(target, quarantineDir string) string {
	encoded := strings.TrimPrefix(target, string(filepath.Separator))
	encoded = strings.ReplaceAll(encoded, string(filepath.Separator), "-")
	return filepath.Join(quarantineDir, "<timestamp>-"+encoded)
}
