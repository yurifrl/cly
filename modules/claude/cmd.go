package claude

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	agentsession "github.com/yurifrl/cly/modules/agent-session"
	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/session"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:                "claude",
		Aliases:            []string{"c"},
		Short:              "Run Claude Code with session management",
		Long:               "Wraps Claude Code with session naming and Zellij tab integration.\n\nPlain `c` opens the UI set by modules.claude.open in config: acp (default, Tern's ACP agent view via `tern acp claude`), tern (tclaude) or claude. Falls back to claude when it can't start; --cli forces claude.\n\nWhen claude is launched directly and headroom is on PATH, it runs through `headroom wrap claude` (modules.claude.headroom, default true). --no-headroom skips it; --no-wrappers skips all wrappers. In acp/tern modes ANTHROPIC_BASE_URL is pointed at the running headroom proxy (modules.claude.headroom.proxy_url) when reachable. --yolo in acp mode opens bypassPermissions and relies on the `cly claude-yolo-context` SessionStart hook for the safe-autonomous prompt.",
		DisableFlagParsing: true,
		RunE:               run,
	}

	parent.AddCommand(cmd)

	parent.AddCommand(&cobra.Command{
		Use:    "claude-yolo-context",
		Short:  "SessionStart hook: emit the safe-autonomous prompt when CLY_YOLO=1",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE:   runYoloContext,
	})

	parent.AddCommand(&cobra.Command{
		Use:    "claude-tern-pane",
		Short:  "SessionStart/SessionEnd hook: record the claude session of a Tern pane for restore",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE:   runTernPane,
	})

	parent.AddCommand(&cobra.Command{
		Use:   "claude-rename <name>",
		Short: "Rename the current Claude session and Zellij tab",
		Args:  cobra.ExactArgs(1),
		RunE:  runRename,
	})
}

// runYoloContext is a Claude Code SessionStart hook. `c --yolo` in acp
// mode sets CLY_YOLO=1 because tern acp can't append a system prompt;
// this injects the safe-autonomous prompt as session context instead.
func runYoloContext(cmd *cobra.Command, args []string) error {
	if os.Getenv("CLY_YOLO") != "1" {
		return nil
	}
	out := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": session.SafeAutonomousPrompt(),
		},
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
}

func runRename(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := session.ValidateName(name); err != nil {
		return err
	}
	sess := &session.Session{Name: name}
	_ = sess.RenameZellijTab()
	fmt.Printf("renamed: %s\n", name)
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	p := ParseArgs(args)
	wrap := session.WithHeadroom(headroomEnabled(p))
	restore := session.WithTernRestore(ternRestoreFlags(p), claudeArgs(p))

	if p.Anonymous {
		fmt.Println("🥷 Anonymous mode")
		return session.ExecClaudeAnonymous(p.PassArgs, wrap)
	}

	if p.ContinueSession != "" {
		return restoreSession(p.ContinueSession, wrap, restore)
	}

	if p.SessionName != "" {
		return resumeOrCreateSession(p.SessionName, wrap, restore)
	}

	sess, err := session.Initialize(p.Name)
	if err != nil {
		return err
	}

	fmt.Printf("🏷️  Session: %s\n", sess.Name)

	_ = sess.RenameZellijTab()

	// Plain `c` / `c --yolo` launch the UI picked by modules.claude.open;
	// it only returns if it can't start, then fall back to claude. Extra
	// claude args or --cli → claude.
	proxy := session.WithAnthropicBaseURL(headroomProxyURL(p))
	extra := p.PassArgs
	if p.Yolo {
		extra = extra[len(session.YoloArgs()):]
	}
	acpArgs, acpOK := ternACPArgs(extra)
	if !p.PlainCLI {
		switch openMode() {
		case "acp":
			if acpOK {
				_ = sess.ExecTernACP(p.Yolo, acpArgs, session.WithTaskListID(p.TaskListID), proxy)
			}
		case "tern":
			if len(extra) != 0 {
				break
			}
			_ = sess.ExecTclaude(p.Yolo, session.WithTaskListID(p.TaskListID), proxy)
		}
	}

	return sess.ExecClaude(p.PassArgs, session.WithTaskListID(p.TaskListID), wrap, restore)
}

// claudeArgs are the claude args the user gave, without the ones --yolo
// injects (ParseArgs puts those first).
func claudeArgs(p ParsedArgs) []string {
	if p.Yolo {
		return p.PassArgs[len(session.YoloArgs()):]
	}
	return p.PassArgs
}

// ternRestoreFlags are the wrapper flags the claude-cli-restore Tern
// plugin replays (after `--cli`) when it resumes this claude.
func ternRestoreFlags(p ParsedArgs) []string {
	var flags []string
	if p.Yolo {
		flags = append(flags, "--yolo")
	}
	if p.NoWrappers {
		flags = append(flags, "--no-wrappers")
	}
	if p.NoHeadroom {
		flags = append(flags, "--no-headroom")
	}
	return flags
}

// ternACPArgs translates claude flags that `tern acp claude` has an
// equivalent for (-r/--resume [id], -c/--continue, --model <m>). ok is
// false when any other arg is present, so the caller falls back to claude.
func ternACPArgs(args []string) (out []string, ok bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		hasVal := i+1 < len(args) && !strings.HasPrefix(args[i+1], "-")
		switch {
		case a == "-r" || a == "--resume":
			if hasVal {
				out = append(out, "--resume", args[i+1])
				i++
			} else {
				out = append(out, "--sessions") // picker
			}
		case a == "-c" || a == "--continue":
			out = append(out, "--continue")
		case a == "--model" && hasVal:
			out = append(out, "--model", args[i+1])
			i++
		default:
			return nil, false
		}
	}
	return out, true
}

// headroomEnabled reports whether claude should launch through
// `headroom wrap claude`: modules.claude.headroom.enabled (default true), unless
// --no-wrappers or --no-headroom is given. The launch itself only wraps
// when the headroom binary is on PATH.
func headroomEnabled(p ParsedArgs) bool {
	if p.NoWrappers || p.NoHeadroom {
		return false
	}
	return config.GetBoolWithDefault("modules.claude.headroom.enabled", true)
}

// headroomProxyURL returns the Headroom proxy base URL for launchers that
// spawn claude themselves (tern acp, tclaude), where `headroom wrap` can't
// be used. The proxy is expected to be running under a process manager;
// nothing is started here. Returns "" when headroom is disabled or the
// proxy isn't reachable, so claude then talks to Anthropic directly.
// URL comes from modules.claude.headroom.proxy_url (default http://127.0.0.1:8787).
func headroomProxyURL(p ParsedArgs) string {
	if !headroomEnabled(p) {
		return ""
	}
	raw := config.GetString("modules.claude.headroom.proxy_url")
	if raw == "" {
		raw = "http://127.0.0.1:8787"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	conn, err := net.DialTimeout("tcp", u.Host, 300*time.Millisecond)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  headroom proxy not reachable at %s; continuing without it\n", u.Host)
		return ""
	}
	_ = conn.Close()
	return raw
}

// openMode returns how plain `c` opens Claude Code, from
// modules.claude.open in config:
//
//	acp    Tern's ACP agent view (`tern acp claude`, claude-agent-acp) — default
//	tern   tclaude (tern-agents-ui)
//	claude the plain Claude Code TUI
func openMode() string {
	switch m := config.GetString("modules.claude.open"); m {
	case "tern", "claude":
		return m
	default:
		return "acp"
	}
}

func restoreSession(name string, opts ...session.ExecOption) error {
	sessions, err := agentsession.Load(agentsession.FilePath())
	if err != nil {
		return err
	}

	entry := agentsession.FindByName(sessions, name)
	if entry == nil {
		return fmt.Errorf("session %q not found", name)
	}

	fmt.Printf("Resuming session: %s\n", name)

	if err := os.Chdir(entry.Path); err != nil {
		return fmt.Errorf("chdir %s: %w", entry.Path, err)
	}

	return session.ExecClaude([]string{"-r", entry.ID}, opts...)
}

func resumeOrCreateSession(name string, opts ...session.ExecOption) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	sessions, err := agentsession.Load(agentsession.FilePath())
	if err != nil {
		return err
	}

	entry := agentsession.FindByName(sessions, name)
	if entry != nil {
		fmt.Printf("📂 Resuming session: %s (id=%s)\n", name, entry.ID)
		return session.ExecClaude([]string{"-r", entry.ID}, opts...)
	}

	// Create new session
	fmt.Printf("✨ Creating new session: %s\n", name)

	sessionID := uuid.New().String()

	sessions["claude:"+name] = agentsession.Entry{
		ID:       sessionID,
		Name:     name,
		Provider: "claude",
		Path:     path,
	}
	if err := agentsession.Save(agentsession.FilePath(), sessions); err != nil {
		return err
	}

	return session.ExecClaude([]string{"--session-id", sessionID}, opts...)
}
