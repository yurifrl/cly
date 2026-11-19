package claude

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

// Tern keeps pane programs alive across window restarts in its session
// daemon, but when the daemon itself restarts (a crash, an update it can't
// take in place) every pane comes back as a fresh shell. Tern resumes its
// own `tern acp` and omp chats then; a plain claude TUI it can't see.
//
// `cly claude-tern-pane` is a SessionStart/SessionEnd hook that keeps one
// record per Tern pane running a claude launched by `c` (CLY_TERN_RESTORE
// set): <ternPaneDir>/<TERN_PANE>.json. The claude-cli-restore Tern plugin
// reads it from its `spawn` hook and has the respawned shell run
// `cly claude --cli [flags] --resume <session_id>`, or the original
// `cly claude --cli [flags] [args]` when the session has no transcript yet.

// ternPaneRecord is the file the claude-cli-restore plugin reads.
type ternPaneRecord struct {
	SessionID string `json:"session_id"`
	// Transcript is where claude writes the session; it only exists once a
	// message was sent, and only then can the session be resumed.
	Transcript string   `json:"transcript_path"`
	Cwd        string   `json:"cwd"`
	Flags      []string `json:"flags"`
	Args       []string `json:"args"`
	Updated    string   `json:"updated"`
}

type ternHookInput struct {
	SessionID     string `json:"session_id"`
	Transcript    string `json:"transcript_path"`
	Cwd           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	Reason        string `json:"reason"`
}

// ternPaneMaxAge prunes records of panes that closed with claude still
// running (no SessionEnd we act on); pane ids are never reused.
const ternPaneMaxAge = 30 * 24 * time.Hour

func ternPaneDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "cly", "tern-claude")
}

func runTernPane(cmd *cobra.Command, args []string) error {
	raw := os.Getenv(session.TernRestoreEnv)
	pane := os.Getenv("TERN_PANE")
	if raw == "" || pane == "" || strings.ContainsAny(pane, `/\.`) {
		return nil
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil
	}
	var in ternHookInput
	if json.Unmarshal(data, &in) != nil || in.SessionID == "" {
		return nil
	}
	// A claude run from claude's Bash tool inherits the env but has no
	// terminal; only the pane's interactive claude owns one.
	if !parentClaudeHasTTY() {
		return nil
	}

	dir := ternPaneDir()
	path := filepath.Join(dir, pane+".json")

	switch in.HookEventName {
	case "SessionStart":
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil
		}
		pruneTernPanes(dir)
		var how session.TernRestore
		if json.Unmarshal([]byte(raw), &how) != nil {
			// claude started by an older cly: space-joined flags.
			how = session.TernRestore{Flags: strings.Fields(raw)}
		}
		rec := ternPaneRecord{
			SessionID:  in.SessionID,
			Transcript: in.Transcript,
			Cwd:        in.Cwd,
			Flags:      how.Flags,
			Args:       how.Args,
			Updated:    time.Now().UTC().Format(time.RFC3339),
		}
		out, _ := json.Marshal(rec)
		tmp := path + ".tmp"
		if os.WriteFile(tmp, out, 0o644) == nil {
			_ = os.Rename(tmp, path)
		}
	case "SessionEnd":
		// "other" is how claude ends when its terminal goes away (the daemon
		// died): keep the record. Any other reason (/exit, ctrl-d, /clear,
		// /resume, logout) is a deliberate end; /clear and /resume write a
		// new record on the SessionStart that follows.
		if in.Reason == "other" {
			return nil
		}
		var rec ternPaneRecord
		if b, err := os.ReadFile(path); err == nil && json.Unmarshal(b, &rec) == nil && rec.SessionID == in.SessionID {
			_ = os.Remove(path)
		}
	}
	return nil
}

func pruneTernPanes(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err == nil && time.Since(info.ModTime()) > ternPaneMaxAge {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// parentClaudeHasTTY walks up from the hook past the shells claude runs
// hooks through, and reports whether the first other ancestor (claude)
// has a controlling terminal.
func parentClaudeHasTTY() bool {
	pid := os.Getppid()
	for range 8 {
		out, err := exec.Command("ps", "-o", "ppid=,tty=,comm=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return false
		}
		f := strings.Fields(string(out))
		if len(f) < 3 {
			return false
		}
		switch filepath.Base(strings.Join(f[2:], " ")) {
		case "sh", "bash", "zsh", "dash", "fish", "-sh", "-bash", "-zsh", "-fish":
			ppid, err := strconv.Atoi(f[0])
			if err != nil || ppid <= 1 {
				return false
			}
			pid = ppid
			continue
		}
		return f[1] != "??" && f[1] != "?"
	}
	return false
}
