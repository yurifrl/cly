package dotfiles

import (
	"fmt"
	"os/user"
	"runtime"
	"strings"

	"github.com/yurifrl/cly/pkg/style"
)

// Target is an inline `@target` gate. Each field is an allow-list; an empty
// field means "no constraint on this axis".
//
//	./file -> ~/file @target user=yuri os=darwin
//
// Values are comma-separated (OR within an axis); axes are ANDed together.
type Target struct {
	Users   []string
	OSes    []string
	Arches  []string
	LineNum int
	set     bool
}

// currentContext returns (username, GOOS, GOARCH). It is a var so tests can
// substitute a fixed machine identity.
var currentContext = func() (string, string, string) {
	name := ""
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	return name, runtime.GOOS, runtime.GOARCH
}

// currentUsername is the OS-detected user.
func currentUsername() string {
	name, _, _ := currentContext()
	return name
}

// effectiveUsername is the identity used both to pick dotfiles.<user>.conf and
// to evaluate a `@target user=` constraint, so the two always agree. --user
// overrides the detected user, which is what makes `dotfiles --user bob`
// reproduce bob's setup on any machine.
func effectiveUsername() string {
	if userFlag != "" {
		return userFlag
	}
	return currentUsername()
}

// gateSuffix renders a matched inline gate for output, so a machine-specific
// entry shows why it applied instead of looking unconditional. Empty for
// ungated entries.
func gateSuffix(gate string) string {
	if gate == "" {
		return ""
	}
	return " " + style.SubtleStyle.Render("("+gate+")")
}

// splitInlineGate peels a trailing `@target ...` gate off a directive line,
// returning the directive and the gate text. Only a gate introduced by
// whitespace counts, so a quoted or embedded "@target" inside a command stays
// part of the command.
func splitInlineGate(line string) (directive, gate string, found bool) {
	idx := strings.LastIndex(line, " @target")
	if idx < 0 {
		return line, "", false
	}
	after := line[idx+len(" @target"):]
	if after != "" && !strings.HasPrefix(after, " ") {
		return line, "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}

func parseTarget(line string, lineNum int) (Target, error) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "@target"))
	if rest == "" {
		return Target{}, fmt.Errorf("@target requires at least one of user=, os=, arch=")
	}

	t := Target{LineNum: lineNum, set: true}
	for _, field := range strings.Fields(rest) {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[1]) == "" {
			return Target{}, fmt.Errorf("invalid @target token %q (expected key=value)", field)
		}
		vals := splitCSV(kv[1])
		switch kv[0] {
		case "user":
			t.Users = vals
		case "os":
			t.OSes = vals
		case "arch":
			t.Arches = vals
		default:
			return Target{}, fmt.Errorf("unknown @target key %q (want user, os, or arch)", kv[0])
		}
	}
	return t, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// GateReason returns "" when the target permits the current machine, or a
// human-readable reason why it does not. An unset target permits everything.
func (t Target) GateReason() string {
	if !t.set {
		return ""
	}
	_, goos, arch := currentContext()
	usr := effectiveUsername()
	if len(t.Users) > 0 && !contains(t.Users, usr) {
		return fmt.Sprintf("config targets user=%s but current user is %q", strings.Join(t.Users, ","), usr)
	}
	if len(t.OSes) > 0 && !contains(t.OSes, goos) {
		return fmt.Sprintf("config targets os=%s but current os is %q", strings.Join(t.OSes, ","), goos)
	}
	if len(t.Arches) > 0 && !contains(t.Arches, arch) {
		return fmt.Sprintf("config targets arch=%s but current arch is %q", strings.Join(t.Arches, ","), arch)
	}
	return ""
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
