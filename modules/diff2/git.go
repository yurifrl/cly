package diff2

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// FileStatus describes the git status of a changed file.
type FileStatus string

const (
	StatusAdded     FileStatus = "added"
	StatusModified  FileStatus = "modified"
	StatusDeleted   FileStatus = "deleted"
	StatusRenamed   FileStatus = "renamed"
	StatusCopied    FileStatus = "copied"
	StatusUntracked FileStatus = "untracked"
	StatusUnknown   FileStatus = "unknown"
)

// File is one entry in the working-tree diff.
type File struct {
	Path      string     `json:"path"`
	OldPath   string     `json:"oldPath,omitempty"` // set on rename/copy
	Status    FileStatus `json:"status"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Binary    bool       `json:"binary"`
}

// Line is one line of a diff hunk.
type Line struct {
	Kind string `json:"kind"` // "context", "add", "del", "meta"
	Old  int    `json:"old"`  // 0 when N/A
	New  int    `json:"new"`  // 0 when N/A
	Text string `json:"text"`
}

// Hunk is a contiguous group of diff lines for a file.
type Hunk struct {
	Header string `json:"header"`
	Lines  []Line `json:"lines"`
}

// FileDiff is the full diff for one file.
type FileDiff struct {
	Path   string `json:"path"`
	Binary bool   `json:"binary"`
	Hunks  []Hunk `json:"hunks"`
}

// Gitter executes git commands. Replace in tests.
type Gitter interface {
	Run(args ...string) ([]byte, error)
}

// execGit shells out to the real git binary in the current working dir.
type execGit struct{}

func (execGit) Run(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	// `git diff` exits 1 when diffs exist — not a failure.
	// Return captured stdout + the error; callers decide how to interpret.
	if err != nil && out.Len() == 0 {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, errBuf.String())
	}
	return out.Bytes(), nil
}

// ErrNotRepo indicates we are not inside a git working tree.
var ErrNotRepo = errors.New("diff2: not inside a git repository")

// IsRepo returns nil if the current dir is inside a git working tree.
func IsRepo(g Gitter) error {
	out, err := g.Run("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return ErrNotRepo
	}
	if strings.TrimSpace(string(out)) != "true" {
		return ErrNotRepo
	}
	return nil
}

// ListChangedFiles returns files changed vs HEAD (staged + unstaged) plus
// untracked files. Binary files are included with Binary=true.
func ListChangedFiles(g Gitter) ([]File, error) {
	statusOut, err := g.Run("diff", "HEAD", "--name-status", "-z")
	if err != nil {
		// empty repo (no HEAD) — still handle via ls-files only
		statusOut = nil
	}

	files := parseNameStatusZ(string(statusOut))

	// merge numstat for +/- counts
	if numOut, err := g.Run("diff", "HEAD", "--numstat"); err == nil {
		applyNumstat(files, string(numOut))
	}

	// append untracked
	if ut, err := g.Run("ls-files", "--others", "--exclude-standard", "-z"); err == nil {
		for _, p := range splitZ(string(ut)) {
			if p == "" {
				continue
			}
			files = append(files, &File{
				Path:   p,
				Status: StatusUntracked,
			})
		}
	}

	// detect binary via full diff scan — cheap marker check
	if binOut, err := g.Run("diff", "HEAD", "--numstat"); err == nil {
		markBinary(files, string(binOut))
	}

	out := make([]File, 0, len(files))
	for _, f := range files {
		out = append(out, *f)
	}
	return out, nil
}

// DiffFile returns the unified diff hunks for one path.
// Returns (nil, binary=true) when the file is binary.
// For untracked files, emits the whole file as "add" lines so the UI
// can render something useful.
func DiffFile(g Gitter, path string) (*FileDiff, error) {
	// Try tracked diff first.
	out, err := g.Run("diff", "HEAD", "--", path)
	if err != nil {
		return nil, err
	}
	raw := string(out)
	if isBinaryDiff(raw) {
		return &FileDiff{Path: path, Binary: true, Hunks: []Hunk{}}, nil
	}

	// Untracked → HEAD diff is empty. Fall back to diff against /dev/null
	// which renders the whole file as additions.
	if strings.TrimSpace(raw) == "" {
		if untrackedOut, err := g.Run("diff", "--no-index", "--", "/dev/null", path); err == nil || len(untrackedOut) > 0 {
			raw = string(untrackedOut)
			if isBinaryDiff(raw) {
				return &FileDiff{Path: path, Binary: true, Hunks: []Hunk{}}, nil
			}
		}
	}

	hunks := parseHunks(raw)
	if hunks == nil {
		hunks = []Hunk{}
	}
	return &FileDiff{Path: path, Hunks: hunks}, nil
}

// parseNameStatusZ parses `git diff --name-status -z` output.
// Format per entry: <status>\t<path>\0  OR  R<score>\t<old>\0<new>\0
func parseNameStatusZ(s string) []*File {
	var out []*File
	parts := strings.Split(s, "\x00")
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if p == "" {
			continue
		}
		// status + TAB + path1 (+ TAB path2 for renames already split above)
		tabIdx := strings.Index(p, "\t")
		if tabIdx < 0 {
			// continuation — rename's 2nd path
			continue
		}
		code := p[:tabIdx]
		path := p[tabIdx+1:]

		f := &File{Path: path}
		switch {
		case code == "A":
			f.Status = StatusAdded
		case code == "M":
			f.Status = StatusModified
		case code == "D":
			f.Status = StatusDeleted
		case strings.HasPrefix(code, "R"):
			f.Status = StatusRenamed
			f.OldPath = path
			// next part is the new path
			if i+1 < len(parts) {
				i++
				f.Path = parts[i]
			}
		case strings.HasPrefix(code, "C"):
			f.Status = StatusCopied
			f.OldPath = path
			if i+1 < len(parts) {
				i++
				f.Path = parts[i]
			}
		default:
			f.Status = StatusUnknown
		}
		out = append(out, f)
	}
	return out
}

// applyNumstat parses `git diff --numstat` and fills +/- on matching files.
// Lines: "<add>\t<del>\t<path>" or "-\t-\t<path>" for binary.
func applyNumstat(files []*File, s string) {
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) != 3 {
			continue
		}
		path := cols[2]
		for _, f := range files {
			if f.Path == path || f.OldPath == path {
				if cols[0] == "-" {
					f.Binary = true
				} else {
					fmt.Sscanf(cols[0], "%d", &f.Additions)
					fmt.Sscanf(cols[1], "%d", &f.Deletions)
				}
				break
			}
		}
	}
}

// markBinary sets Binary=true on files with "-\t-" numstat entry.
func markBinary(files []*File, s string) {
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := sc.Text()
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) == 3 && cols[0] == "-" && cols[1] == "-" {
			for _, f := range files {
				if f.Path == cols[2] || f.OldPath == cols[2] {
					f.Binary = true
				}
			}
		}
	}
}

// splitZ splits NUL-separated strings, dropping empty trailer.
func splitZ(s string) []string {
	s = strings.TrimRight(s, "\x00")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x00")
}

// isBinaryDiff returns true when git reports "Binary files ... differ".
func isBinaryDiff(raw string) bool {
	return strings.Contains(raw, "\nBinary files ") ||
		strings.HasPrefix(raw, "Binary files ")
}

// parseHunks parses a unified-diff body into hunks.
// Input is the raw output of `git diff HEAD -- <path>`.
func parseHunks(raw string) []Hunk {
	var hunks []Hunk
	var cur *Hunk
	oldLine, newLine := 0, 0

	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "@@"):
			if cur != nil {
				hunks = append(hunks, *cur)
			}
			cur = &Hunk{Header: line}
			oldLine, newLine = parseHunkHeader(line)
		case cur == nil:
			// skip diff preamble ("diff --git", "index", "---", "+++")
			continue
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			cur.Lines = append(cur.Lines, Line{Kind: "add", New: newLine, Text: line[1:]})
			newLine++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			cur.Lines = append(cur.Lines, Line{Kind: "del", Old: oldLine, Text: line[1:]})
			oldLine++
		case strings.HasPrefix(line, " "):
			cur.Lines = append(cur.Lines, Line{Kind: "context", Old: oldLine, New: newLine, Text: line[1:]})
			oldLine++
			newLine++
		case strings.HasPrefix(line, "\\"):
			// "\ No newline at end of file"
			cur.Lines = append(cur.Lines, Line{Kind: "meta", Text: line})
		}
	}
	if cur != nil {
		hunks = append(hunks, *cur)
	}
	return hunks
}

// parseHunkHeader extracts the starting old/new line numbers from
// a hunk header like "@@ -42,7 +42,12 @@ some context".
func parseHunkHeader(h string) (oldStart, newStart int) {
	// find "-X,Y +A,B"
	a := strings.Index(h, "-")
	b := strings.Index(h, "+")
	if a < 0 || b < 0 {
		return 1, 1
	}
	oldPart := h[a+1 : b-1]
	end := strings.Index(h[b:], " ")
	if end < 0 {
		end = len(h) - b
	}
	newPart := h[b+1 : b+end]

	parseOne := func(s string) int {
		// s is "X" or "X,Y"
		if comma := strings.Index(s, ","); comma >= 0 {
			s = s[:comma]
		}
		n := 0
		fmt.Sscanf(s, "%d", &n)
		if n == 0 {
			n = 1
		}
		return n
	}
	return parseOne(oldPart), parseOne(newPart)
}
