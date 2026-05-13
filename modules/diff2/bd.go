package diff2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// BdRunner executes bd commands. Replace in tests.
type BdRunner interface {
	Run(args ...string) ([]byte, []byte, error)
}

type execBd struct{}

func (execBd) Run(args ...string) ([]byte, []byte, error) {
	cmd := exec.Command("bd", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// Errors for bd integration states.
var (
	ErrBdMissing  = errors.New("diff2: bd CLI not installed")
	ErrBdNoDB     = errors.New("diff2: no beads database (run `bd init`)")
	ErrBdBadInput = errors.New("diff2: bad bead input")
)

// BeadRequest is the form payload from the UI.
type BeadRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`     // bug|feature|task|chore|decision
	Priority    string   `json:"priority"` // 0-4 or P0-P4
	Context     string   `json:"context,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// BeadResponse is what we return to the UI.
type BeadResponse struct {
	ID string `json:"id"`
}

// ListLabels returns all known labels from `bd label list-all --json`.
// Returns nil (with nil error) when bd is missing or no DB exists —
// callers degrade gracefully.
func ListLabels(b BdRunner) ([]string, error) {
	stdout, stderr, err := b.Run("label", "list-all", "--json")
	if err != nil {
		return classifyBdErrAsList(stderr, err)
	}
	// output may be "[]" or a JSON array of strings / objects
	raw := bytes.TrimSpace(stdout)
	if len(raw) == 0 || string(raw) == "[]" {
		return []string{}, nil
	}

	// try []string first
	var labels []string
	if err := json.Unmarshal(raw, &labels); err == nil {
		return labels, nil
	}
	// fallback: array of objects with "label" or "name"
	var objs []map[string]any
	if err := json.Unmarshal(raw, &objs); err == nil {
		for _, o := range objs {
			if s, ok := o["label"].(string); ok {
				labels = append(labels, s)
			} else if s, ok := o["name"].(string); ok {
				labels = append(labels, s)
			}
		}
		return labels, nil
	}
	return []string{}, nil
}

// CreateBead shells out to `bd create` and returns the new bead ID.
func CreateBead(b BdRunner, req BeadRequest) (*BeadResponse, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("%w: title required", ErrBdBadInput)
	}
	args := []string{"create", req.Title, "--json"}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}
	if req.Type != "" {
		args = append(args, "--type", req.Type)
	}
	if req.Priority != "" {
		args = append(args, "--priority", normalizePriority(req.Priority))
	}
	if req.Context != "" {
		args = append(args, "--context", req.Context)
	}
	if len(req.Labels) > 0 {
		args = append(args, "--labels", strings.Join(req.Labels, ","))
	}

	stdout, stderr, err := b.Run(args...)
	if err != nil {
		return nil, classifyBdErrAsCreate(stderr, err)
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout), &parsed); err != nil {
		return nil, fmt.Errorf("diff2: parse bd output: %w", err)
	}
	if parsed.ID == "" {
		return nil, fmt.Errorf("diff2: bd returned empty id: %s", string(stdout))
	}
	return &BeadResponse{ID: parsed.ID}, nil
}

// normalizePriority accepts "P2", "p2", "2" → "2".
func normalizePriority(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "P")
	p = strings.TrimPrefix(p, "p")
	if _, err := strconv.Atoi(p); err == nil {
		return p
	}
	return "2"
}

// classifyBdErrAsList distinguishes bd-missing vs no-db vs other.
// Returns (nil, nil) for missing/no-db so label list degrades to empty.
func classifyBdErrAsList(stderr []byte, err error) ([]string, error) {
	if isBdMissing(err) {
		return []string{}, nil
	}
	if isNoDB(stderr) {
		return []string{}, nil
	}
	return nil, fmt.Errorf("diff2: bd label list-all: %w: %s", err, string(stderr))
}

func classifyBdErrAsCreate(stderr []byte, err error) error {
	if isBdMissing(err) {
		return ErrBdMissing
	}
	if isNoDB(stderr) {
		return ErrBdNoDB
	}
	return fmt.Errorf("diff2: bd create failed: %w: %s", err, string(stderr))
}

func isBdMissing(err error) bool {
	if err == nil {
		return false
	}
	var e *exec.Error
	if errors.As(err, &e) {
		return true
	}
	// fallback on substring
	return strings.Contains(err.Error(), "executable file not found")
}

func isNoDB(stderr []byte) bool {
	s := string(stderr)
	return strings.Contains(s, "no beads database") ||
		strings.Contains(s, "no beads database found")
}
