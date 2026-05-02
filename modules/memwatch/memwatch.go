package memwatch

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Sample is a single read of macOS memory_pressure output.
type Sample struct {
	FreePercent  int    // "System-wide memory free percentage"
	PressureLvl  string // "normal" | "warn" | "critical" | "unknown"
	Raw          string
}

var (
	freePctRe  = regexp.MustCompile(`System-wide memory free percentage:\s*(\d+)`)
	pressureRe = regexp.MustCompile(`(?i)memory pressure:\s*(\w+)`)
)

// Read runs `memory_pressure` and parses its output.
func Read(ctx context.Context) (*Sample, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(cctx, "memory_pressure").Output()
	if err != nil {
		return nil, fmt.Errorf("run memory_pressure: %w", err)
	}

	s := &Sample{FreePercent: -1, PressureLvl: "unknown", Raw: string(out)}

	scanner := bufio.NewScanner(strings.NewReader(s.Raw))
	for scanner.Scan() {
		line := scanner.Text()
		if m := freePctRe.FindStringSubmatch(line); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				s.FreePercent = n
			}
		}
		if m := pressureRe.FindStringSubmatch(line); m != nil {
			s.PressureLvl = strings.ToLower(m[1])
		}
	}

	// Fallback pressure classification from free %
	if s.PressureLvl == "unknown" && s.FreePercent >= 0 {
		switch {
		case s.FreePercent <= 10:
			s.PressureLvl = "critical"
		case s.FreePercent <= 25:
			s.PressureLvl = "warn"
		default:
			s.PressureLvl = "normal"
		}
	}

	return s, nil
}
