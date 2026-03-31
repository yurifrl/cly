package pitree

import (
	"fmt"
	"strings"
	"time"
)

// RenderTree returns the plain-text tree representation.
func RenderTree(nodes []WorkspaceNode) string {
	var b strings.Builder
	for _, ws := range nodes {
		b.WriteString(fmt.Sprintf("  %s\n", ws.Name))
		for i, s := range ws.Sessions {
			branch := "├──"
			if i == len(ws.Sessions)-1 {
				branch = "└──"
			}
			size := formatSize(s.SizeBytes)
			b.WriteString(fmt.Sprintf("    %s %s  %s  %s\n",
				branch, s.SessionID, size, s.StartedAt))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func formatSize(bytes int64) string {
	kb := bytes / 1024
	if kb >= 1024 {
		return fmt.Sprintf("%.1fMB", float64(kb)/1024)
	}
	return fmt.Sprintf("%dKB", kb)
}

func countSessions(tree []WorkspaceNode) int {
	n := 0
	for _, ws := range tree {
		n += len(ws.Sessions)
	}
	return n
}

// fmtSnapTime formats a snapshot's timestamp in a human-readable way for the picker.
func fmtSnapTime(s Snapshot) string {
	t := s.CreatedAt
	now := time.Now()
	switch {
	case t.Format("2006-01-02") == now.Format("2006-01-02"):
		return "Today " + t.Format("15:04:05")
	case t.Format("2006-01-02") == now.AddDate(0, 0, -1).Format("2006-01-02"):
		return "Yesterday " + t.Format("15:04:05")
	default:
		return t.Format("Mon Jan 2  15:04:05")
	}
}

