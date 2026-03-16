package zs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type runtimeOptions struct {
	DryRun           bool
	Debug            bool
	ForceSessionMode bool
	ForceTabMode     bool
	Stderr           *os.File
	Stdout           *os.File
}

var runtimeCfg = runtimeOptions{
	Stderr: os.Stderr,
	Stdout: os.Stdout,
}

func configureRuntime(dryRun, debug, forceSessionMode, forceTabMode bool) {
	runtimeCfg.DryRun = dryRun
	runtimeCfg.Debug = debug
	runtimeCfg.ForceSessionMode = forceSessionMode
	runtimeCfg.ForceTabMode = forceTabMode
	runtimeCfg.Stderr = os.Stderr
	runtimeCfg.Stdout = os.Stdout
}

func debugf(format string, args ...any) {
	if !runtimeCfg.Debug {
		return
	}
	fmt.Fprintf(runtimeCfg.Stderr, "zs debug: "+format+"\n", args...)
}

func printDryRun(args ...string) {
	fmt.Fprintf(runtimeCfg.Stdout, "DRY RUN: %s\n", shellJoin(args))
}

func shellJoin(args []string) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = shellQuote(arg)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return strings.ContainsRune(" \t\n\r\"'\\$&;|<>()[]{}*?!#~`", r)
	}) == -1 {
		return s
	}
	return strconv.Quote(s)
}

func envBool(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func summarizeStrings(items []string, limit int) string {
	if len(items) == 0 {
		return "[]"
	}
	if limit <= 0 || len(items) <= limit {
		return fmt.Sprintf("%v", items)
	}
	return fmt.Sprintf("%v ... (+%d more)", items[:limit], len(items)-limit)
}
