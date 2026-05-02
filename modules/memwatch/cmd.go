package memwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/cmux"
	"github.com/yurifrl/cly/pkg/notify"
)

// Config is the memwatch module configuration.
type Config struct {
	Enabled          bool
	Interval         time.Duration
	ThresholdPercent int      // alert when free% <= this
	CriticalPercent  int      // critical alert threshold
	Cooldown         time.Duration
	AlertOnPressure  []string // levels that trigger: warn, critical
	Title            string
	Message          string
	UseCmux          bool
	UseDesktop       bool
	UseZellij        bool
	Sound            string
	TopN             int
	ProcThresholdMB  int64 // alert when a single process exceeds this RSS
	ProcGrowthMB     int64 // alert when a process grows by this much since last alert
	IncludeTopInAlert bool
}

func defaultConfig() Config {
	return Config{
		Enabled:          true,
		Interval:         30 * time.Second,
		ThresholdPercent: 20,
		CriticalPercent:  10,
		Cooldown:         5 * time.Minute,
		AlertOnPressure:  []string{"warn", "critical"},
		Title:            "🧠 Low Memory",
		Message:          "Free: ${FREE}% — Pressure: ${PRESSURE}",
		UseCmux:          true,
		UseDesktop:       true,
		UseZellij:        false,
		Sound:            "Basso",
		TopN:             5,
		ProcThresholdMB:  1500,
		ProcGrowthMB:     500,
		IncludeTopInAlert: true,
	}
}

func loadConfig() Config {
	cfg := defaultConfig()
	c := pkgconfig.Get()
	if c == nil {
		return cfg
	}
	m, ok := c.Modules["memwatch"]
	if !ok {
		return cfg
	}
	if v, ok := m["enabled"].(bool); ok {
		cfg.Enabled = v
	}
	if v, ok := m["interval"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Interval = d
		}
	}
	if v, ok := m["threshold_percent"].(int); ok {
		cfg.ThresholdPercent = v
	}
	if v, ok := m["critical_percent"].(int); ok {
		cfg.CriticalPercent = v
	}
	if v, ok := m["cooldown"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Cooldown = d
		}
	}
	if v, ok := m["alert_on_pressure"].([]interface{}); ok {
		cfg.AlertOnPressure = nil
		for _, x := range v {
			if s, ok := x.(string); ok {
				cfg.AlertOnPressure = append(cfg.AlertOnPressure, strings.ToLower(s))
			}
		}
	}
	if v, ok := m["title"].(string); ok {
		cfg.Title = v
	}
	if v, ok := m["message"].(string); ok {
		cfg.Message = v
	}
	if v, ok := m["use_cmux"].(bool); ok {
		cfg.UseCmux = v
	}
	if v, ok := m["use_desktop"].(bool); ok {
		cfg.UseDesktop = v
	}
	if v, ok := m["use_zellij"].(bool); ok {
		cfg.UseZellij = v
	}
	if v, ok := m["sound"].(string); ok {
		cfg.Sound = v
	}
	if v, ok := m["top_n"].(int); ok {
		cfg.TopN = v
	}
	if v, ok := m["process_threshold_mb"].(int); ok {
		cfg.ProcThresholdMB = int64(v)
	}
	if v, ok := m["process_growth_mb"].(int); ok {
		cfg.ProcGrowthMB = int64(v)
	}
	if v, ok := m["include_top_in_alert"].(bool); ok {
		cfg.IncludeTopInAlert = v
	}
	return cfg
}

func Register(parent *cobra.Command) {
	root := &cobra.Command{
		Use:   "memwatch",
		Short: "macOS memory pressure watcher with notifications",
		Long: `Polls macOS memory_pressure and sends notifications when free memory
drops below a configured threshold or pressure state becomes warn/critical.

Designed to be launched at login via ~/Dotfiles/dotfiles.conf:
  @startup memwatch keepalive -- cly memwatch run`,
	}

	root.AddCommand(
		newTopCmd(),
		newPiCmd(),
		&cobra.Command{
			Use:   "check",
			Short: "One-shot memory pressure check (prints JSON)",
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := Read(cmd.Context())
				if err != nil {
					return err
				}
				b, _ := json.MarshalIndent(map[string]any{
					"free_percent": s.FreePercent,
					"pressure":     s.PressureLvl,
				}, "", "  ")
				fmt.Println(string(b))
				return nil
			},
		},
		&cobra.Command{
			Use:   "run",
			Short: "Run the memory watcher loop (blocks; designed for @startup keepalive)",
			RunE:  runLoop,
		},
		newNotifyCmd(),
		&cobra.Command{
			Use:   "clear",
			Short: "Clear the cmux memwatch sidebar status badge",
			RunE: func(cmd *cobra.Command, args []string) error {
				if !cmux.Available() {
					fmt.Fprintln(os.Stderr, "memwatch: not inside a cmux session, nothing to clear")
					return nil
				}
				return cmux.ClearStatus(cmd.Context(), "memwatch")
			},
		},
		&cobra.Command{
			Use:   "restart",
			Short: "Kill the running memwatch daemon (dotfiles @startup keepalive relaunches it with the new binary)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return Restart(cmd.Context())
			},
		},
	)

	parent.AddCommand(root)
}

func newPiCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "pi",
		Short: "List running pi instances with memory and working directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			asJSON, _ := cmd.Flags().GetBool("json")
			procs, err := PiProcesses(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(procs, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			if len(procs) == 0 {
				fmt.Println("No running pi instances.")
				return nil
			}
			var total int64
			fmt.Println("PI INSTANCES")
			for _, p := range procs {
				total += p.RSSKB
				name := strings.Join(p.SessionNames, " │ ")
				if name == "" {
					name = "(no session name)"
				}
				ws := p.Workspace
				if ws == "" {
					ws = p.Label
				}
				fmt.Printf("  %-7d  %-20s %10s  %s\n",
					p.PID, truncate(ws, 20), FormatSize(p.RSSKB), truncate(name, 90))
			}
			fmt.Printf("  %-7s  %-20s %10s\n", "", "TOTAL", FormatSize(total))
			return nil
		},
	}
	c.Flags().Bool("json", false, "output as JSON")
	return c
}

func newTopCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "top",
		Short: "Show top memory-hungry processes (aggregated by app)",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, _ := cmd.Flags().GetInt("n")
			asJSON, _ := cmd.Flags().GetBool("json")
			procs, err := TopProcesses(cmd.Context(), n)
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(procs, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			fmt.Println("TOP MEMORY")
			for _, p := range procs {
				fmt.Printf("  %-40s %10s\n", truncate(p.Name, 40), FormatSize(p.RSSKB))
			}
			return nil
		},
	}
	c.Flags().IntP("n", "n", 5, "number of processes to show")
	c.Flags().Bool("json", false, "output as JSON")
	return c
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func runLoop(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	if !cfg.Enabled {
		fmt.Fprintln(os.Stderr, "memwatch: disabled in config, exiting")
		return nil
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	fmt.Fprintf(os.Stderr, "memwatch: running (interval=%s, threshold=%d%%, critical=%d%%, cooldown=%s)\n",
		cfg.Interval, cfg.ThresholdPercent, cfg.CriticalPercent, cfg.Cooldown)

	var lastAlert time.Time
	var lastLevel string
	// procAlerts tracks per-process RSS (KB) at last alert, for growth-based outlier detection.
	procAlerts := make(map[string]int64)

	t := time.NewTicker(cfg.Interval)
	defer t.Stop()

	tick := func() {
		s, err := Read(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "memwatch: read error: %v\n", err)
			return
		}

		// Always snapshot top processes (cheap: single ps call).
		procs, _ := TopProcesses(ctx, cfg.TopN)

		shouldAlert, severity := classify(cfg, s)

		// Keep cmux sidebar status in sync every tick: set when alerting,
		// clear when healthy. This prevents stale orange "RAM 15%" badges.
		syncCmuxStatus(ctx, cfg, s, shouldAlert)

		// Detect per-process outliers independently of global pressure.
		if outliers := detectProcOutliers(cfg, procs, procAlerts); len(outliers) > 0 {
			if err := sendProcAlert(ctx, cfg, s, outliers, procs); err != nil {
				fmt.Fprintf(os.Stderr, "memwatch: proc notify error: %v\n", err)
			}
			for _, p := range outliers {
				procAlerts[p.Name] = p.RSSKB
			}
		}

		if !shouldAlert {
			lastLevel = s.PressureLvl
			return
		}

		now := time.Now()
		escalated := lastLevel != "critical" && severity == "critical"
		if !escalated && now.Sub(lastAlert) < cfg.Cooldown {
			return
		}

		if err := sendAlert(ctx, cfg, s, procs, false); err != nil {
			fmt.Fprintf(os.Stderr, "memwatch: notify error: %v\n", err)
			return
		}
		lastAlert = now
		lastLevel = severity
	}

	tick() // immediate first check
	for {
		select {
		case <-ctx.Done():
			// Best-effort: clear any lingering status badge before exit.
			cleanCtx, cancelClean := context.WithTimeout(context.Background(), 2*time.Second)
			if cfg.UseCmux && cmux.Available() {
				_ = cmux.ClearStatus(cleanCtx, "memwatch")
			}
			cancelClean()
			return nil
		case <-t.C:
			tick()
		}
	}
}

// detectProcOutliers returns processes that cross the absolute threshold or grew
// significantly since the last per-process alert.
func detectProcOutliers(cfg Config, procs []Proc, last map[string]int64) []Proc {
	if cfg.ProcThresholdMB <= 0 && cfg.ProcGrowthMB <= 0 {
		return nil
	}
	var outliers []Proc
	for _, p := range procs {
		mb := p.RSSKB / 1024
		over := cfg.ProcThresholdMB > 0 && mb >= cfg.ProcThresholdMB
		prev := last[p.Name]
		grew := cfg.ProcGrowthMB > 0 && prev > 0 && (p.RSSKB-prev)/1024 >= cfg.ProcGrowthMB
		// Only re-alert on absolute threshold if it grew since last alert too.
		if prev > 0 && over && !grew {
			continue
		}
		if over || grew {
			outliers = append(outliers, p)
		}
	}
	return outliers
}

func sendProcAlert(ctx context.Context, cfg Config, s *Sample, outliers, top []Proc) error {
	names := make([]string, 0, len(outliers))
	for _, p := range outliers {
		names = append(names, fmt.Sprintf("%s (%s)", p.Name, FormatSize(p.RSSKB)))
	}
	title := "🚨 Memory Outlier"
	msg := strings.Join(names, ", ")
	if cfg.IncludeTopInAlert && len(top) > 0 {
		msg += "\n\nTop:\n" + formatTop(top)
	}
	return dispatch(ctx, cfg, s, title, msg)
}

func formatTop(procs []Proc) string {
	var b strings.Builder
	for _, p := range procs {
		fmt.Fprintf(&b, "  %s — %s\n", truncate(p.Name, 40), FormatSize(p.RSSKB))
	}
	return strings.TrimRight(b.String(), "\n")
}

func classify(cfg Config, s *Sample) (alert bool, severity string) {
	severity = "normal"
	switch {
	case s.FreePercent >= 0 && s.FreePercent <= cfg.CriticalPercent:
		return true, "critical"
	case s.FreePercent >= 0 && s.FreePercent <= cfg.ThresholdPercent:
		return true, "warn"
	}
	for _, level := range cfg.AlertOnPressure {
		if strings.EqualFold(level, s.PressureLvl) {
			return true, s.PressureLvl
		}
	}
	return false, severity
}

func expand(tpl string, s *Sample) string {
	r := strings.NewReplacer(
		"${FREE}", fmt.Sprintf("%d", s.FreePercent),
		"${PRESSURE}", s.PressureLvl,
	)
	return os.ExpandEnv(r.Replace(tpl))
}

func sendAlert(ctx context.Context, cfg Config, s *Sample, top []Proc, force bool) error {
	title := expand(cfg.Title, s)
	msg := expand(cfg.Message, s)
	if cfg.IncludeTopInAlert && len(top) > 0 {
		msg += "\n\nTop:\n" + formatTop(top)
	}
	_ = force
	return dispatch(ctx, cfg, s, title, msg)
}

// syncCmuxStatus keeps the cmux sidebar badge in sync with current memory
// state. When alerting: show "RAM N%" with orange (warn) or red (critical).
// When healthy: clear the status so stale badges don't linger.
func syncCmuxStatus(ctx context.Context, cfg Config, s *Sample, alerting bool) {
	if !cfg.UseCmux || !cmux.Available() {
		return
	}
	if !alerting || s.FreePercent < 0 {
		_ = cmux.ClearStatus(ctx, "memwatch")
		return
	}
	icon := "exclamationmark.triangle"
	color := "#E67E22"
	if s.PressureLvl == "critical" || s.FreePercent <= cfg.CriticalPercent {
		color = "#C0392B"
	}
	_ = cmux.SetStatus(ctx, "memwatch", fmt.Sprintf("RAM %d%%", s.FreePercent),
		cmux.WithIcon(icon), cmux.WithColor(color))
}

func dispatch(ctx context.Context, cfg Config, s *Sample, title, msg string) error {
	sent := false

	if cfg.UseCmux && cmux.BinaryAvailable() {
		if err := cmux.Notify(ctx, title, msg); err == nil {
			sent = true
		} else {
			fmt.Fprintf(os.Stderr, "memwatch: cmux notify failed: %v\n", err)
		}
	}

	if cfg.UseDesktop && !sent {
		n := notify.New("memwatch", cfg.UseZellij, false, false, "")
		_ = n.Send(ctx, notify.Notification{
			Title:   title,
			Message: msg,
			Sound:   cfg.Sound,
			Group:   "cly-memwatch",
		})
	}

	fmt.Fprintf(os.Stderr, "memwatch: ALERT %s — %s\n", title, strings.ReplaceAll(msg, "\n", " | "))
	return nil
}
