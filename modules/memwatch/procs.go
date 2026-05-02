package memwatch

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Proc is a single aggregated process entry.
type Proc struct {
	Name   string // display name (pretty)
	Path   string // representative full path
	RSSKB  int64  // aggregated resident set size (KB)
	Count  int    // number of processes aggregated
}

// TopProcesses returns the top N memory-hungry processes, aggregated by app name.
// Uses `ps -axo rss=,comm=` — aggregation collapses multi-process apps (Chrome, etc.).
func TopProcesses(ctx context.Context, n int) ([]Proc, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "rss=,comm=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	agg := make(map[string]*Proc)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			continue
		}
		rss, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64)
		if err != nil {
			continue
		}
		path := strings.TrimSpace(fields[1])
		name := prettyName(path)
		if p, ok := agg[name]; ok {
			p.RSSKB += rss
			p.Count++
		} else {
			agg[name] = &Proc{Name: name, Path: path, RSSKB: rss, Count: 1}
		}
	}

	procs := make([]Proc, 0, len(agg))
	for _, p := range agg {
		procs = append(procs, *p)
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].RSSKB > procs[j].RSSKB })
	if n > 0 && len(procs) > n {
		procs = procs[:n]
	}
	return procs, nil
}

// prettyName extracts a human-readable app name from a process path.
//   /Applications/Google Chrome.app/Contents/MacOS/Google Chrome -> Google Chrome
//   /Applications/Obsidian.app/Contents/Frameworks/Obsidian Helper (Renderer).app/... -> Obsidian Helper (Renderer)
//   /usr/local/bin/pi -> pi
func prettyName(path string) string {
	// Prefer the deepest *.app bundle name.
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasSuffix(parts[i], ".app") {
			return strings.TrimSuffix(parts[i], ".app")
		}
	}
	return filepath.Base(path)
}

// FormatSize renders KB as "1.3 GB" / "730 MB" / "512 KB".
func FormatSize(kb int64) string {
	switch {
	case kb >= 1024*1024:
		return fmt.Sprintf("%.1f GB", float64(kb)/1024.0/1024.0)
	case kb >= 1024:
		return fmt.Sprintf("%d MB", kb/1024)
	default:
		return fmt.Sprintf("%d KB", kb)
	}
}
