package ompsummary

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Digest is the structured last-command view of a transcript tail: the opening
// request, the latest substantive user message, the latest assistant text, and
// the latest tool activity.
type Digest struct {
	FirstUser  string
	LastUser   string
	LastUserAt float64 // unix seconds of LastUser, 0 unknown
	LastAssist string
	LastTool   string // "<tool>: <intent>"
	Exit       int    // latest known exit code, ExitUnknown when none
	Turns      int    // user+assistant messages seen in the tail
}

var (
	exitRe    = regexp.MustCompile(`exited with code (\d+)`)
	exitJSON  = regexp.MustCompile(`"exitCode":\s*(\d+)`)
	wsRegex   = regexp.MustCompile(`\s+`)
	intentCap = 120
)

// DigestTail reads the last maxTail bytes of a transcript (dropping the
// partial first line, the file can be multi-MB) and returns both the
// structured Digest and the same flattened role-line tail the summarizer
// prompt consumes, capped at maxChars on a line boundary.
//
// Line shape (verified): {"type":"message","timestamp":"RFC3339","message":{
// "role","content":[{"type":"text","text"}]}}, plus {"type":"custom",
// "customType":"tool_execution_*","data":{"toolName","intent"}}.
func DigestTail(path string, maxTail, maxChars int) (Digest, string, error) {
	var d Digest
	d.Exit = ExitUnknown
	if path == "" {
		return d, "", os.ErrInvalid
	}
	fi, err := os.Stat(path)
	if err != nil {
		return d, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return d, "", err
	}
	defer f.Close()

	read := int64(maxTail)
	if fi.Size() < read {
		read = fi.Size()
	}
	buf := make([]byte, read)
	if _, err := f.ReadAt(buf, fi.Size()-read); err != nil {
		return d, "", err
	}
	lines := strings.Split(string(buf), "\n")
	if read < fi.Size() && len(lines) > 0 {
		lines = lines[1:] // drop partial first line
	}

	var parts []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var e struct {
			Type       string `json:"type"`
			CustomType string `json:"customType"`
			Timestamp  string `json:"timestamp"`
			Message    *struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Data struct {
				ToolName string `json:"toolName"`
				Intent   string `json:"intent"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			continue
		}
		at := parseRFC3339(e.Timestamp)
		switch {
		case e.Type == "custom" && e.Data.ToolName != "":
			d.LastTool = capRunes(e.Data.ToolName+": "+e.Data.Intent, intentCap)
			parts = append(parts, "→ "+e.Data.ToolName+": "+e.Data.Intent)
		case e.Type == "message" && e.Message != nil:
			role := e.Message.Role
			if role == "" {
				role = "user"
			}
			text := extractText(e.Message.Content)
			switch role {
			case "user":
				if text == "" {
					continue
				}
				d.Turns++
				if d.FirstUser == "" {
					d.FirstUser = capRunes(text, intentCap)
				}
				d.LastUser = capRunes(text, intentCap)
				d.LastUserAt = at
				parts = append(parts, "user: "+text)
			case "assistant":
				if text == "" {
					continue
				}
				d.Turns++
				d.LastAssist = capRunes(text, intentCap)
				parts = append(parts, "assistant: "+text)
			case "toolResult":
				if text != "" {
					parts = append(parts, "→ "+capRunes(text, 160))
				}
				if code, ok := lastExitCode(text); ok {
					d.Exit = code
				}
			default:
				if text != "" {
					parts = append(parts, role+": "+text)
				}
			}
		}
	}
	out := strings.Join(parts, "\n")
	if len(out) > maxChars {
		out = out[len(out)-maxChars:]
		if i := strings.IndexByte(out, '\n'); i > 0 {
			out = out[i+1:] // start on a line boundary
		}
	}
	return d, out, nil
}

// lastExitCode extracts an exit code from tool-result text, preferring the
// human "exited with code N" phrasing over embedded JSON fields. Reports the
// LAST match so the newest command wins.
func lastExitCode(text string) (int, bool) {
	for _, re := range []*regexp.Regexp{exitRe, exitJSON} {
		if ms := re.FindAllStringSubmatch(text, -1); len(ms) > 0 {
			n, err := strconv.Atoi(ms[len(ms)-1][1])
			if err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

func parseRFC3339(s string) float64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return float64(t.Unix())
}

func capRunes(s string, n int) string {
	r := []rune(wsRegex.ReplaceAllString(strings.TrimSpace(s), " "))
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}
